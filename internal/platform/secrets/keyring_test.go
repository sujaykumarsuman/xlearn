package secrets

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	k := make([]byte, MasterKeySize)
	if _, err := rand.Read(k); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return k
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

// TestSealADRoundTrip is the baseline: the same associated data gets the secret back.
func TestSealADRoundTrip(t *testing.T) {
	c, err := NewCipher(testKey(t))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	secret := []byte("sk-ant-api03-not-a-real-key")
	ad := CoachKeyAD("acct-1", "anthropic")

	encKey, encDataKey, err := c.SealAD(secret, ad)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Contains(encKey, secret) || bytes.Contains(encDataKey, secret) {
		t.Fatal("sealed output contains the plaintext")
	}
	got, err := c.OpenAD(encKey, encDataKey, ad)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if !bytes.Equal(got, secret) {
		t.Fatalf("round trip = %q, want %q", got, secret)
	}
}

// TestSealADBindsAccountAndProvider is the whole reason the AD pair exists: a sealed pair
// moved to another account — or to the same account's other provider — must FAIL to open
// rather than silently decrypting into the wrong context. Anyone with database write
// access could otherwise swap two rows' sealed bytes and have the coach bill one learner's
// key for another's chat.
func TestSealADBindsAccountAndProvider(t *testing.T) {
	c, err := NewCipher(testKey(t))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}
	secret := []byte("sk-owner-key")
	ad := CoachKeyAD("acct-1", "anthropic")
	encKey, encDataKey, err := c.SealAD(secret, ad)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}

	for _, tc := range []struct {
		name string
		ad   []byte
	}{
		{"another account", CoachKeyAD("acct-2", "anthropic")},
		{"another provider", CoachKeyAD("acct-1", "openai")},
		{"both swapped", CoachKeyAD("acct-2", "openai")},
		{"no associated data", nil},
		{"truncated ad", []byte("xlearn/coach/key/v1|acct-1")},
		{"wrong ad version", []byte("xlearn/coach/key/v2|acct-1|anthropic")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := c.OpenAD(encKey, encDataKey, tc.ad)
			if !errors.Is(err, ErrDecrypt) {
				t.Fatalf("open with %q = (%q, %v), want ErrDecrypt", tc.ad, got, err)
			}
			if got != nil {
				t.Fatal("a failed open returned plaintext")
			}
		})
	}
}

// TestSealADBindsBothLayers checks the ad is on the data-key wrap too, not just the
// secret. If only the inner layer were bound, an attacker could keep a victim's
// enc_data_key and substitute their own enc_key.
func TestSealADBindsBothLayers(t *testing.T) {
	c, _ := NewCipher(testKey(t))
	adA := CoachKeyAD("acct-1", "anthropic")
	adB := CoachKeyAD("acct-2", "anthropic")

	keyA, dataA, _ := c.SealAD([]byte("secret-a"), adA)
	keyB, dataB, _ := c.SealAD([]byte("secret-b"), adB)

	// Mix and match across the two pairs: every combination must fail.
	for _, tc := range []struct {
		name            string
		encKey, encData []byte
		ad              []byte
	}{
		{"A key with B data under A's ad", keyA, dataB, adA},
		{"A key with B data under B's ad", keyA, dataB, adB},
		{"B key with A data under A's ad", keyB, dataA, adA},
		{"B key with A data under B's ad", keyB, dataA, adB},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := c.OpenAD(tc.encKey, tc.encData, tc.ad); !errors.Is(err, ErrDecrypt) {
				t.Fatalf("open = %v, want ErrDecrypt", err)
			}
		})
	}
}

// TestLegacyPairIsUnbound pins that Seal/Open keep the v1 byte format and pass NO
// associated data. This is what keeps v1.6.0 a valid rollback target (ADR-0034 §3) — if
// the legacy pair ever started carrying ad, every key would become unreadable by the
// previous release and the declared rollback floor would be a lie.
func TestLegacyPairIsUnbound(t *testing.T) {
	c, _ := NewCipher(testKey(t))
	secret := []byte("sk-legacy")

	encKey, encDataKey, err := c.Seal(secret)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	got, err := c.Open(encKey, encDataKey)
	if err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("legacy round trip = (%q, %v)", got, err)
	}
	// The legacy pair must be openable with NO ad — which is exactly what v1.6.0 does.
	if got, err := c.OpenAD(encKey, encDataKey, nil); err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("OpenAD(nil ad) on a legacy pair = (%q, %v); the legacy format changed", got, err)
	}
	// And NOT with ad, which is why a second pair is needed at all.
	if _, err := c.OpenAD(encKey, encDataKey, CoachKeyAD("a", "openai")); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("legacy pair opened with ad = %v, want ErrDecrypt", err)
	}
}

func TestCoachKeyADFormat(t *testing.T) {
	// The exact wire format from ADR-0031 §7. It is persisted implicitly (every stored AD
	// pair is bound to it), so changing it silently would make every key unreadable.
	want := "xlearn/coach/key/v1|11111111-1111-4111-8111-111111111111|anthropic"
	if got := string(CoachKeyAD("11111111-1111-4111-8111-111111111111", "anthropic")); got != want {
		t.Fatalf("CoachKeyAD = %q, want %q", got, want)
	}
}

func TestKeyringSealsUnderActiveAndOpensUnderStamped(t *testing.T) {
	k0, k1 := testKey(t), testKey(t)
	// First entry is active, so rotating means prepending.
	kr, err := ParseKeyring("k1:" + b64(k1) + ",k0:" + b64(k0))
	if err != nil {
		t.Fatalf("parse keyring: %v", err)
	}
	if kr.Active() != "k1" {
		t.Fatalf("active = %q, want k1 (the first entry)", kr.Active())
	}
	if !kr.Has("k0") || !kr.Has("k1") || kr.Has("k2") {
		t.Fatalf("Has: k0=%v k1=%v k2=%v", kr.Has("k0"), kr.Has("k1"), kr.Has("k2"))
	}

	ad := CoachKeyAD("acct-1", "openai")
	secret := []byte("sk-rotated")
	encKey, encData, kekID, err := kr.SealAD(secret, ad)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if kekID != "k1" {
		t.Fatalf("sealed under %q, want the active entry k1", kekID)
	}
	if got, err := kr.OpenAD(encKey, encData, kekID, ad); err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("open under k1 = (%q, %v)", got, err)
	}

	// A pair sealed by the RETIRING key still opens — that is what makes a rotation
	// zero-downtime: the re-wrap job walks rows at its own pace while chat keeps working.
	oldCipher, _ := NewCipher(k0)
	oldKey, oldData, _ := oldCipher.SealAD(secret, ad)
	if got, err := kr.OpenAD(oldKey, oldData, "k0", ad); err != nil || !bytes.Equal(got, secret) {
		t.Fatalf("open under the retiring k0 = (%q, %v)", got, err)
	}
	// Opening under the WRONG entry fails as a decrypt error, not as a wrong-key success.
	if _, err := kr.OpenAD(oldKey, oldData, "k1", ad); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("k0 pair opened under k1 = %v, want ErrDecrypt", err)
	}
}

// TestKeyringUnknownKEK pins that a retired entry is distinguishable from corruption:
// ErrUnknownKEK means "we don't hold that key", which the re-wrap job reports and the
// owner fixes by restoring the entry — not by re-pasting every learner's key.
func TestKeyringUnknownKEK(t *testing.T) {
	kr, err := ParseKeyring("k0:" + b64(testKey(t)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	ad := CoachKeyAD("a", "openai")
	encKey, encData, _, _ := kr.SealAD([]byte("sk"), ad)

	_, err = kr.OpenAD(encKey, encData, "k7", ad)
	if !errors.Is(err, ErrUnknownKEK) {
		t.Fatalf("open under a missing entry = %v, want ErrUnknownKEK", err)
	}
	if errors.Is(err, ErrDecrypt) {
		t.Fatal("ErrUnknownKEK must not read as ErrDecrypt: a retired key is not a corrupt row")
	}
	// An empty kek_id (a row written before 00006) is also "unknown", never a silent
	// fall-through to the active key.
	if _, err := kr.OpenAD(encKey, encData, "", ad); !errors.Is(err, ErrUnknownKEK) {
		t.Fatalf("open under an empty kek id = %v, want ErrUnknownKEK", err)
	}
}

func TestParseKeyringErrors(t *testing.T) {
	good := b64(testKey(t))
	for _, tc := range []struct {
		name string
		spec string
		want error
	}{
		{"empty", "", ErrEmptyKeyring},
		{"whitespace only", "   ", ErrEmptyKeyring},
		{"only commas", ",,,", ErrEmptyKeyring},
		{"no id", good, ErrBadKeyringSpec},
		{"bad id shape", "key0:" + good, ErrBadKeyringSpec},
		{"uppercase id", "K0:" + good, ErrBadKeyringSpec},
		{"id without digits", "k:" + good, ErrBadKeyringSpec},
		{"duplicate id", "k0:" + good + ",k0:" + good, ErrBadKeyringSpec},
		{"short key", "k0:" + base64.StdEncoding.EncodeToString([]byte("too-short")), ErrBadMasterKey},
		{"empty key", "k0:", ErrEmptyMasterKey},
	} {
		t.Run(tc.name, func(t *testing.T) {
			kr, err := ParseKeyring(tc.spec)
			if !errors.Is(err, tc.want) {
				t.Fatalf("ParseKeyring = (%v, %v), want %v", kr, err, tc.want)
			}
			if kr != nil {
				t.Fatal("a rejected spec returned a keyring")
			}
			// A spec is key material: it must never appear in the error text.
			if err != nil && tc.spec != "" && strings.Contains(err.Error(), good) {
				t.Fatalf("error echoes key material: %q", err.Error())
			}
		})
	}
}

func TestParseKeyringAcceptsHexAndWhitespace(t *testing.T) {
	// ParseMasterKey's encodings all work per entry, and surrounding whitespace in a
	// mounted file is tolerated — a SOPS secret should not need exact formatting.
	key := testKey(t)
	hexKey := ""
	for _, b := range key {
		const digits = "0123456789abcdef"
		hexKey += string(digits[b>>4]) + string(digits[b&0x0f])
	}
	kr, err := ParseKeyring(" k1: " + hexKey + " , k0: " + b64(testKey(t)) + " \n")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if kr.Active() != "k1" || !kr.Has("k0") {
		t.Fatalf("active=%q has(k0)=%v", kr.Active(), kr.Has("k0"))
	}
	// The hex entry really decoded to the same key.
	direct, _ := NewCipher(key)
	ad := CoachKeyAD("a", "openai")
	encKey, encData, _ := direct.SealAD([]byte("sk"), ad)
	if _, err := kr.OpenAD(encKey, encData, "k1", ad); err != nil {
		t.Fatalf("hex entry did not decode to the expected key: %v", err)
	}
}

func TestKeyringOfSingleEntry(t *testing.T) {
	// The production shape while COACH_MASTER_KEYS is unset: one entry, k0, over the
	// existing master key. New pairs are stamped k0 so a later rotation has something to
	// rotate FROM.
	c, _ := NewCipher(testKey(t))
	kr := KeyringOf("k0", c)
	if kr.Active() != "k0" || len(kr.IDs()) != 1 {
		t.Fatalf("active=%q ids=%v", kr.Active(), kr.IDs())
	}
	ad := CoachKeyAD("a", "anthropic")
	encKey, encData, kekID, err := kr.SealAD([]byte("sk"), ad)
	if err != nil || kekID != "k0" {
		t.Fatalf("seal = (%q, %v)", kekID, err)
	}
	// The same cipher opens it directly — the keyring adds naming, not a format change.
	if got, err := c.OpenAD(encKey, encData, ad); err != nil || string(got) != "sk" {
		t.Fatalf("direct open = (%q, %v)", got, err)
	}
}

func TestNewKeyringRejectsBadInput(t *testing.T) {
	k := testKey(t)
	if _, err := NewKeyring("k0", nil); !errors.Is(err, ErrEmptyKeyring) {
		t.Fatalf("empty map = %v, want ErrEmptyKeyring", err)
	}
	if _, err := NewKeyring("k9", map[string][]byte{"k0": k}); !errors.Is(err, ErrUnknownKEK) {
		t.Fatalf("active not in map = %v, want ErrUnknownKEK", err)
	}
	if _, err := NewKeyring("bad", map[string][]byte{"bad": k}); !errors.Is(err, ErrBadKeyringSpec) {
		t.Fatalf("bad id = %v, want ErrBadKeyringSpec", err)
	}
}
