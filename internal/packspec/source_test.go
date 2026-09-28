package packspec_test

import (
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/packspec"
)

// Every value here is synthetic (hand-made; never derived from xlearn-evalpack).
var (
	h1 = "sha256:" + strings.Repeat("a1", 32)
	h2 = "sha256:" + strings.Repeat("b2", 32)
	h3 = "sha256:" + strings.Repeat("c3", 32)
)

func TestDecodeRoot(t *testing.T) {
	r, err := packspec.DecodeRoot([]byte(`{"format_major": 0, "version": "0.1.0"}`))
	if err != nil || !r.SupportedFormat() || r.Validate() != nil {
		t.Fatalf("scaffold root: %+v %v", r, err)
	}
	for _, bad := range []string{`{"format_major": 0, "version": "0.1.0", "x": 1}`, `{"format_major": 0} {}`} {
		if _, err := packspec.DecodeRoot([]byte(bad)); err == nil {
			t.Errorf("DecodeRoot(%s) accepted", bad)
		}
	}
	r2, _ := packspec.DecodeRoot([]byte(`{"format_major": 7, "version": "v1"}`))
	if r2.SupportedFormat() || r2.Validate() == nil {
		t.Fatal("format 7 / version v1 accepted")
	}
}

func TestDecodeAndValidateItem(t *testing.T) {
	good := `{
	  "item": "900",
	  "accepts_contract_hashes": ["` + h1 + `"],
	  "wrong": [{"file": "submissions/wrong/slow.go", "expect": "TLE", "category": "complexity_misjudged"},
	            {"file": "submissions/wrong/off.go", "expect": "WA"}],
	  "keys": {"p-structure": "keys/p-structure.json"},
	  "review": {"tests": "2026-10-09"}
	}`
	it, err := packspec.DecodeItem([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if err := it.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !it.Stamped() {
		t.Fatal("stamped pack reported unstamped")
	}
	if got := strings.Join(it.Files(), ","); got != "keys/p-structure.json,submissions/wrong/off.go,submissions/wrong/slow.go" {
		t.Fatalf("Files = %s", got)
	}
	if _, err := packspec.DecodeItem([]byte(`{"item": "900", "accepts_contract_hashes": [], "answers": {}}`)); err == nil {
		t.Fatal("an unknown field decoded")
	}

	for name, tc := range map[string]struct {
		it   packspec.Item
		want string
	}{
		"no hash":      {packspec.Item{Item: "900"}, "needs the live contract hash"},
		"three hashes": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1, h2, h3}}, "at most 2"},
		"bad hash":     {packspec.Item{Item: "900", AcceptsContractHashes: []string{"sha256:abc"}}, "not sha256:<64 hex>"},
		"dup hash":     {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1, h1}}, "duplicate"},
		"bad expect": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1},
			Wrong: []packspec.Wrong{{File: "submissions/wrong/a.go", Expect: "PASS"}}}, "expect"},
		"escaping file": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1},
			Wrong: []packspec.Wrong{{File: "../x.go", Expect: "WA"}}}, "clean relative path"},
		"file outside submissions": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1},
			Wrong: []packspec.Wrong{{File: "tests/a.go", Expect: "WA"}}}, "under submissions/"},
		"dotfile": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1},
			Wrong: []packspec.Wrong{{File: "submissions/.hidden.go", Expect: "WA"}}}, "dotfile"},
		"key outside keys": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1},
			Keys: map[string]string{"est": "anchors/est.json"}}, "under keys/"},
		"bad date": {packspec.Item{Item: "900", AcceptsContractHashes: []string{h1},
			Review: &packspec.Review{Tests: "2026-13-01"}}, "ISO date"},
	} {
		err := tc.it.Validate()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Validate = %v, want %q", name, err, tc.want)
		}
	}
}
