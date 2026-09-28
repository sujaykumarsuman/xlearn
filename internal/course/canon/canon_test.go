package canon_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
)

var update = flag.Bool("update", false, "rewrite testdata/content.golden.json")

// The content-hash golden: m1-01's valid item fixtures, each resolved with the same
// fixed sections, hashed. A change here is a canon change: every stored content_hash
// moves once at the next seed (harmless: the seed rewrites sections on every run).
const goldenFile = "testdata/content.golden.json"

var fixtures = []string{"valid-self.json", "valid-code.json", "valid-probes.json", "valid-structured.json"}

func fixtureSections() []course.Section {
	return []course.Section{
		{Stage: "attempt", Order: 1, Kind: "summary", BodyMD: "Given an array, return the answer → fast."},
		{Stage: "hint", Order: 1, Kind: "key_observation", BodyMD: "Sort first & sweep <two> pointers."},
		{Stage: "solution", Order: 2, Kind: "code", Language: "go", Code: "func f() int {\n    return 1\n}"},
		{Stage: "solution", Order: 1, Kind: "approach", BodyMD: "One pass."},
	}
}

func resolved(t *testing.T, name string) *course.ResolvedItem {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", "items", name))
	if err != nil {
		t.Fatal(err)
	}
	it, err := course.DecodeItem(b)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return &course.ResolvedItem{Item: *it, Sections: fixtureSections()}
}

func mustHash(t *testing.T, it *course.ResolvedItem) string {
	t.Helper()
	h, err := canon.ContentHash(it)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestContentHashGolden(t *testing.T) {
	got := map[string]string{}
	for _, name := range fixtures {
		got[name] = mustHash(t, resolved(t, name))
	}
	enc, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	enc = append(enc, '\n')
	if *update {
		if err := os.WriteFile(goldenFile, enc, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("read %s (run with -update): %v", goldenFile, err)
	}
	if !bytes.Equal(want, enc) {
		t.Fatalf("content hashes changed (a canon change moves every stored content_hash):\nwant %s\ngot  %s", want, enc)
	}
}

func TestContentHashShape(t *testing.T) {
	h := mustHash(t, resolved(t, "valid-self.json"))
	if !strings.HasPrefix(h, "sha256:") || len(h) != len("sha256:")+64 {
		t.Fatalf("hash %q is not sha256:<64 hex>", h)
	}
}

// Formatting never moves the hash: whitespace, key order, the sections' file order and
// number spelling in open sample values.
func TestContentHashIgnoresFormatting(t *testing.T) {
	base := mustHash(t, resolved(t, "valid-code.json"))

	// Re-serialize item.json with other whitespace and reversed key order.
	raw, err := os.ReadFile(filepath.Join("..", "testdata", "items", "valid-code.json"))
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	respelled, err := json.MarshalIndent(generic, "", "\t\t")
	if err != nil {
		t.Fatal(err)
	}
	it, err := course.DecodeItem(respelled)
	if err != nil {
		t.Fatal(err)
	}
	secs := fixtureSections()
	secs[0], secs[3] = secs[3], secs[0]
	if got := mustHash(t, &course.ResolvedItem{Item: *it, Sections: secs}); got != base {
		t.Errorf("reformatted item: hash %s, want %s", got, base)
	}

	// Number spelling inside a sample: 1000 ≡ 1e3 ≡ 1000.0, and key order inside objects.
	a := resolved(t, "valid-code.json")
	b := resolved(t, "valid-code.json")
	if len(a.Item.Parts) == 0 || len(a.Item.Parts[0].Config.Samples) == 0 {
		t.Fatal("valid-code.json has no sample to respell")
	}
	a.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"b": 1000, "a": [0.000001, -0]}`)
	b.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"a":[1e-6,0],"b":1e3}`)
	if ha, hb := mustHash(t, a), mustHash(t, b); ha != hb {
		t.Errorf("respelled sample numbers moved the hash: %s vs %s", ha, hb)
	}
}

// Any real edit moves the hash.
func TestContentHashSeesEdits(t *testing.T) {
	base := mustHash(t, resolved(t, "valid-self.json"))
	for name, edit := range map[string]func(*course.ResolvedItem){
		"title":        func(r *course.ResolvedItem) { r.Item.Title += "!" },
		"status":       func(r *course.ResolvedItem) { r.Item.Status = "retired" },
		"section body": func(r *course.ResolvedItem) { r.Sections[0].BodyMD += " " },
		"code":         func(r *course.ResolvedItem) { r.Sections[2].Code += "\n" },
		"language":     func(r *course.ResolvedItem) { r.Sections[2].Language = "cpp" },
		"new section": func(r *course.ResolvedItem) {
			r.Sections = append(r.Sections, course.Section{Stage: "hint", Order: 2, Kind: "pattern", BodyMD: "x"})
		},
	} {
		r := resolved(t, "valid-self.json")
		edit(r)
		if got := mustHash(t, r); got == base {
			t.Errorf("%s edit did not move the hash", name)
		}
	}
}

func TestContentHashDoesNotMutate(t *testing.T) {
	r := resolved(t, "valid-code.json")
	r.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"b": 1e3}`)
	before := append([]course.Section(nil), r.Sections...)
	mustHash(t, r)
	if string(r.Item.Parts[0].Config.Samples[0].Expected) != `{"b": 1e3}` {
		t.Error("ContentHash rewrote the caller's sample")
	}
	for i := range before {
		if r.Sections[i] != before[i] {
			t.Fatal("ContentHash reordered the caller's sections")
		}
	}
}

func TestNormalizeJSON(t *testing.T) {
	for in, want := range map[string]string{
		`[1, 2 ,3]`:                  `[1,2,3]`,
		`-0`:                         `0`,
		`1e3`:                        `1000`,
		`1.5e0`:                      `1.5`,
		`0.000001`:                   `1e-06`,
		`{"z":{"b":2,"a":1},"a":[]}`: `{"a":[],"z":{"a":1,"b":2}}`,
		`"<&>"`:                      `"<&>"`,
		`9223372036854775807`:        `9223372036854775807`,
	} {
		got, err := canon.NormalizeJSON(json.RawMessage(in))
		if err != nil {
			t.Errorf("NormalizeJSON(%s): %v", in, err)
			continue
		}
		if string(got) != want {
			t.Errorf("NormalizeJSON(%s) = %s, want %s", in, got, want)
		}
	}
	for _, bad := range []string{`9223372036854775808`, `[1,`, `1 2`} {
		if _, err := canon.NormalizeJSON(json.RawMessage(bad)); err == nil {
			t.Errorf("NormalizeJSON(%s) = nil error, want one", bad)
		}
	}
}
