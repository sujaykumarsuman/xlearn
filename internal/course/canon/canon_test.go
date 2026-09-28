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

var update = flag.Bool("update", false, "rewrite testdata/vectors.golden.json")

// The golden vectors (canon@1): m1-01's original item fixtures plus the class-mode one,
// each resolved with the same fixed sections, and the DSA manifest. A change here is a
// canon change: every stored content_hash moves at the next seed and every pack's
// accepts_contract_hashes goes stale. That is canon@2, never an edit — CI never runs
// -update.
const goldenFile = "testdata/vectors.golden.json"

var fixtures = []string{"valid-self.json", "valid-code.json", "valid-probes.json", "valid-structured.json", "valid-class.json"}

const dsaManifest = "../../../curriculum/courses/dsa/course.json"

func fixtureSections() []course.Section {
	return []course.Section{
		{Stage: "attempt", Order: 1, Kind: "summary", BodyMD: "Given an array, return the answer → fast."},
		{Stage: "hint", Order: 1, Kind: "key_observation", BodyMD: "Sort first & sweep <two> pointers."},
		{Stage: "solution", Order: 2, Kind: "code", Language: "go", Code: "func f() int {\n    return 1\n}"},
		{Stage: "solution", Order: 1, Kind: "approach", BodyMD: "One pass."},
	}
}

func readItem(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", "items", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func decode(t *testing.T, b []byte) *course.Item {
	t.Helper()
	it, err := course.DecodeItem(b)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func resolved(t *testing.T, name string) *course.ResolvedItem {
	t.Helper()
	return &course.ResolvedItem{Item: *decode(t, readItem(t, name)), Sections: fixtureSections()}
}

func mustHash(t *testing.T, it *course.ResolvedItem) string {
	t.Helper()
	h, err := canon.ContentHash(it)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func mustContract(t *testing.T, it *course.Item) string {
	t.Helper()
	h, err := canon.ContractHash(it)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func manifest(t *testing.T) *course.Manifest {
	t.Helper()
	b, err := os.ReadFile(dsaManifest)
	if err != nil {
		t.Fatal(err)
	}
	m, err := course.DecodeManifest(b)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

type vector struct {
	ContentHash   string  `json:"content_hash,omitempty"`
	ContractHash  *string `json:"contract_hash,omitempty"`
	ContractBytes string  `json:"contract_bytes,omitempty"`
	PolicyVersion string  `json:"policy_version,omitempty"`
}

func TestGoldenVectors(t *testing.T) {
	got := map[string]vector{}
	for _, name := range fixtures {
		r := resolved(t, name)
		cb, err := canon.ContractBytes(&r.Item)
		if err != nil {
			t.Fatal(err)
		}
		ch := mustContract(t, &r.Item)
		v := vector{ContentHash: mustHash(t, r), ContractHash: &ch}
		if ch != "" {
			v.ContractBytes = string(cb)
		}
		got["item:"+name] = v
	}
	pv, err := canon.PolicyVersion(manifest(t))
	if err != nil {
		t.Fatal(err)
	}
	got["manifest:dsa"] = vector{PolicyVersion: pv}

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
		t.Fatalf("canon@1 bytes changed (every stored hash and every pack's accepts_contract_hashes would go stale; that is canon@2):\nwant %s\ngot  %s", want, enc)
	}
}

func TestHashShapeAndDomains(t *testing.T) {
	r := resolved(t, "valid-code.json")
	for _, h := range []string{mustHash(t, r), mustContract(t, &r.Item)} {
		if !strings.HasPrefix(h, "sha256:") || len(h) != len("sha256:")+64 {
			t.Fatalf("hash %q is not sha256:<64 hex>", h)
		}
	}
	b := []byte(`{"a":1}`)
	if canon.Sum(canon.KindContent, b) == canon.Sum(canon.KindContract, b) ||
		canon.Sum(canon.KindContract, b) == canon.Sum(canon.KindPolicy, b) {
		t.Fatal("domain separation: equal bytes hashed under two kinds coincide")
	}
}

func TestPrefix(t *testing.T) {
	h := canon.Sum(canon.KindContract, []byte("x"))
	if p := canon.Prefix(h); len(p) != 12 || !strings.HasPrefix(h, "sha256:"+p) {
		t.Fatalf("Prefix(%s) = %q", h, p)
	}
	if canon.Prefix("") != "" {
		t.Fatal(`Prefix("") must be ""`)
	}
}

// A self-path item (no graded part, no pack-keyed probe) has no contract.
func TestSelfPathHasNoContract(t *testing.T) {
	if h := mustContract(t, decode(t, readItem(t, "valid-self.json"))); h != "" {
		t.Fatalf("self-path ContractHash = %q, want \"\"", h)
	}
	// Ungraded parts and public-keyed probes stay on the self path.
	it := decode(t, readItem(t, "valid-code.json"))
	it.Parts[0].Grading = "none"
	it.Grader = nil
	if !canon.IsSelfPath(it) || mustContract(t, it) != "" {
		t.Fatal("an item whose only part is ungraded must be self-path")
	}
	// A pack-keyed probe needs the pack, so it has a contract.
	if h := mustContract(t, decode(t, readItem(t, "valid-probes.json"))); h == "" {
		t.Fatal("an item with a pack-keyed probe must have a contract")
	}
}

// Formatting never moves either hash: whitespace, key order, the sections' file order,
// absent vs empty, and number spelling in open sample values and checker params.
func TestNonEditsMoveNothing(t *testing.T) {
	for _, name := range fixtures {
		raw := readItem(t, name)
		base := resolved(t, name)
		baseContent, baseContract := mustHash(t, base), mustContract(t, &base.Item)

		// Re-serialize item.json with other whitespace and sorted (not authored) key order.
		var generic map[string]any
		if err := json.Unmarshal(raw, &generic); err != nil {
			t.Fatal(err)
		}
		respelled, err := json.MarshalIndent(generic, "", "\t\t")
		if err != nil {
			t.Fatal(err)
		}
		secs := fixtureSections()
		secs[0], secs[3] = secs[3], secs[0]
		r := &course.ResolvedItem{Item: *decode(t, respelled), Sections: secs}
		if got := mustHash(t, r); got != baseContent {
			t.Errorf("%s reformatted: content_hash %s, want %s", name, got, baseContent)
		}
		if got := mustContract(t, &r.Item); got != baseContract {
			t.Errorf("%s reformatted: contract_hash %s, want %s", name, got, baseContract)
		}
	}

	// Number spelling inside a sample: 1000 ≡ 1e3, 0.000001 ≡ 1e-6, -0 ≡ 0, key order.
	a := resolved(t, "valid-code.json")
	b := resolved(t, "valid-code.json")
	a.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"b": 1000, "a": [0.000001, -0]}`)
	b.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"a":[1e-6,0],"b":1e3}`)
	if ha, hb := mustHash(t, a), mustHash(t, b); ha != hb {
		t.Errorf("respelled sample numbers moved content_hash: %s vs %s", ha, hb)
	}

	// A checker eps spelled 1e-6 vs 0.000001 (and 1E-6), in the authored JSON.
	src := string(readItem(t, "valid-code.json"))
	var hs []string
	for _, eps := range []string{"0.000001", "1e-6", "1E-6", "0.0000010"} {
		j := strings.Replace(src, `"checker": { "name": "exact" }`, `"checker": { "name": "float_abs", "params": { "eps": `+eps+` } }`, 1)
		if j == src {
			t.Fatal("valid-code.json checker not found")
		}
		r := &course.ResolvedItem{Item: *decode(t, []byte(j)), Sections: fixtureSections()}
		hs = append(hs, mustHash(t, r)+" "+mustContract(t, &r.Item))
	}
	for _, h := range hs[1:] {
		if h != hs[0] {
			t.Errorf("a respelled checker eps moved a hash: %v", hs)
			break
		}
	}

	// Absent ≡ empty: {} / [] where the author could leave the field out.
	for what, c := range map[string]struct{ at, empty, absent string }{
		"review {}": {`"review": { "statement": "2026-09-28", "hints": "2026-09-28", "editorial": "2026-09-28" },`,
			`"review": {},`, ``},
		"checker params {}":         {`"checker": { "name": "exact" }`, `"checker": { "name": "exact", "params": {} }`, `"checker": { "name": "exact" }`},
		"links []":                  {`"status": "live",`, `"status": "live", "links": [],`, `"status": "live",`},
		"signature ops []":          {`"returns": "string"`, `"returns": "string", "ops": []`, `"returns": "string"`},
		"provenance inspired_by []": {`"origin": "original",`, `"origin": "original", "inspired_by": [],`, `"origin": "original",`},
	} {
		empty, absent := strings.Replace(src, c.at, c.empty, 1), strings.Replace(src, c.at, c.absent, 1)
		if empty == src && absent == src {
			t.Fatalf("%s: anchor not found in valid-code.json", what)
		}
		ra := &course.ResolvedItem{Item: *decode(t, []byte(empty)), Sections: fixtureSections()}
		rb := &course.ResolvedItem{Item: *decode(t, []byte(absent)), Sections: fixtureSections()}
		if mustHash(t, ra) != mustHash(t, rb) || mustContract(t, &ra.Item) != mustContract(t, &rb.Item) {
			t.Errorf("absent vs empty (%s) moved a hash", what)
		}
	}
}

// The mutation table: a content-only edit moves content_hash only; a contract edit moves
// both.
func TestMutationTable(t *testing.T) {
	type edit struct {
		fixture string
		fn      func(r *course.ResolvedItem)
	}
	contentOnly := map[string]edit{
		"statement prompt (attempt section)": {"valid-code.json", func(r *course.ResolvedItem) { r.Sections[0].BodyMD += " Now faster." }},
		"section body (hint)":                {"valid-code.json", func(r *course.ResolvedItem) { r.Sections[1].BodyMD += "!" }},
		"a sample":                           {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`"fig"`) }},
		"limits.time_ms":                     {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Limits.TimeMS = 2000 }},
		"languages":                          {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Languages = []string{"go", "py"} }},
		"a param name":                       {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Signature.Params[0].Name = "ballots" }},
		"a constraint":                       {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Constraints[0].Len = []int{1, 100000} }},
		"an option label":                    {"valid-structured.json", func(r *course.ResolvedItem) { r.Item.Parts[3].Config.Options[0].Label = "A CDN" }},
		"a field label":                      {"valid-structured.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Fields[0].Label = "Features" }},
		"a probe prompt":                     {"valid-probes.json", func(r *course.ResolvedItem) { r.Item.Revision.Probes[2].PromptMD += " Explain." }},
		"a probe option label":               {"valid-probes.json", func(r *course.ResolvedItem) { r.Item.Revision.Probes[2].Config.Options[0].Label = "Sorting" }},
		"title":                              {"valid-class.json", func(r *course.ResolvedItem) { r.Item.Title += " II" }},
		"a class sample":                     {"valid-class.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Samples[0].Ops[5] = "Place" }},
		"a reference file":                   {"valid-class.json", func(r *course.ResolvedItem) { r.References = map[string]string{"go": "package x\n"} }},
	}
	contract := map[string]edit{
		"a part id":           {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].ID = "code"; r.Item.Grader[0].Inputs[0] = "code" }},
		"a param type":        {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Signature.Params[0].Type = "string[][]" }},
		"the return type":     {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Signature.Returns = "int" }},
		"the harness version": {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Harness = "func-json@2" }},
		"the checker":         {"valid-code.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Checker.Name = "unordered" }},
		"a checker param": {"valid-code.json", func(r *course.ResolvedItem) {
			r.Item.Parts[0].Config.Checker.Params = &course.CheckerParams{Eps: 1e-9}
		}},
		"an option id":          {"valid-structured.json", func(r *course.ResolvedItem) { r.Item.Parts[3].Config.Options[0].ID = "edge" }},
		"a field id":            {"valid-structured.json", func(r *course.ResolvedItem) { r.Item.Parts[1].Config.Fields[0].ID = "read_qps" }},
		"a rubric version":      {"valid-structured.json", func(r *course.ResolvedItem) { r.Item.Grader[1].Rubric = "system-design/hld@2" }},
		"an asset version":      {"valid-structured.json", func(r *course.ResolvedItem) { r.Item.Assets[0] = "system-design/palette@2" }},
		"a probe id":            {"valid-probes.json", func(r *course.ResolvedItem) { r.Item.Revision.Probes[2].ID = "p-dominant" }},
		"a probe option id":     {"valid-probes.json", func(r *course.ResolvedItem) { r.Item.Revision.Probes[2].Config.Options[0].ID = "z" }},
		"a class op":            {"valid-class.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Signature.Ops[1].Name = "Get" }},
		"a class op param type": {"valid-class.json", func(r *course.ResolvedItem) { r.Item.Parts[0].Config.Signature.Ops[0].Params[1].Type = "string" }},
	}
	for name, e := range contentOnly {
		base, edited := resolved(t, e.fixture), resolved(t, e.fixture)
		e.fn(edited)
		if mustHash(t, base) == mustHash(t, edited) {
			t.Errorf("content-only %s: content_hash did not move", name)
		}
		if mustContract(t, &base.Item) != mustContract(t, &edited.Item) {
			t.Errorf("content-only %s: contract_hash moved", name)
		}
	}
	for name, e := range contract {
		base, edited := resolved(t, e.fixture), resolved(t, e.fixture)
		e.fn(edited)
		if mustHash(t, base) == mustHash(t, edited) {
			t.Errorf("contract %s: content_hash did not move", name)
		}
		if mustContract(t, &base.Item) == mustContract(t, &edited.Item) {
			t.Errorf("contract %s: contract_hash did not move", name)
		}
	}

	// Order of set-valued fields is presentation, not contract.
	a := resolved(t, "valid-structured.json")
	b := resolved(t, "valid-structured.json")
	opts := b.Item.Parts[3].Config.Options
	opts[0], opts[1] = opts[1], opts[0]
	b.Item.Parts[0], b.Item.Parts[2] = b.Item.Parts[2], b.Item.Parts[0]
	in := b.Item.Grader[1].Inputs
	in[0], in[2] = in[2], in[0]
	if mustContract(t, &a.Item) != mustContract(t, &b.Item) {
		t.Error("reordering parts, options or step inputs moved contract_hash")
	}
	if mustHash(t, a) == mustHash(t, b) {
		t.Error("reordering parts or options (presentation) did not move content_hash")
	}
}

// decode → Bytes → decode → Bytes is identical, and the contract bytes are stable.
func TestRoundTrip(t *testing.T) {
	for _, name := range fixtures {
		it := decode(t, readItem(t, name))
		b1, err := canon.Bytes(it)
		if err != nil {
			t.Fatal(err)
		}
		it2 := decode(t, b1)
		b2, err := canon.Bytes(it2)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(b1, b2) {
			t.Errorf("%s: Bytes round trip differs:\n%s\n%s", name, b1, b2)
		}
		if mustContract(t, it) != mustContract(t, it2) {
			t.Errorf("%s: contract hash changed across a round trip", name)
		}
	}
}

func TestPolicyVersion(t *testing.T) {
	m := manifest(t)
	pv, err := canon.PolicyVersion(m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(pv, "sha256:") || len(pv) != 71 {
		t.Fatalf("PolicyVersion = %q", pv)
	}
	// Reformatted manifest: same version.
	raw, err := os.ReadFile(dsaManifest)
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatal(err)
	}
	re, _ := json.MarshalIndent(generic, "", "   ")
	m2, err := course.DecodeManifest(re)
	if err != nil {
		t.Fatal(err)
	}
	if pv2, _ := canon.PolicyVersion(m2); pv2 != pv {
		t.Fatalf("a reformatted manifest moved policy_version: %s vs %s", pv2, pv)
	}
	// Absent ≡ empty: an empty mock pool list vs none.
	m3, _ := course.DecodeManifest(raw)
	if m3.Mock != nil {
		m3.Mock.Pools = nil
		m4, _ := course.DecodeManifest(raw)
		m4.Mock.Pools = []string{}
		p3, _ := canon.PolicyVersion(m3)
		p4, _ := canon.PolicyVersion(m4)
		if p3 != p4 {
			t.Fatal("nil vs empty pools moved policy_version")
		}
	}
	// A real edit moves it.
	m.Stages.Attempt.DurationS++
	if pv3, _ := canon.PolicyVersion(m); pv3 == pv {
		t.Fatal("an attempt-duration edit did not move policy_version")
	}
}

func TestContentHashDoesNotMutate(t *testing.T) {
	r := resolved(t, "valid-code.json")
	r.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"b": 1e3}`)
	before := append([]course.Section(nil), r.Sections...)
	mustHash(t, r)
	mustContract(t, &r.Item)
	if string(r.Item.Parts[0].Config.Samples[0].Expected) != `{"b": 1e3}` {
		t.Error("ContentHash rewrote the caller's sample")
	}
	for i := range before {
		if r.Sections[i] != before[i] {
			t.Fatal("ContentHash reordered the caller's sections")
		}
	}
	s := resolved(t, "valid-structured.json")
	first := s.Item.Parts[0].ID
	mustContract(t, &s.Item)
	if s.Item.Parts[0].ID != first {
		t.Fatal("ContractHash reordered the caller's parts")
	}
}

// Inside an open value, an empty array or object is data, never dropped.
func TestSampleEmptiesAreData(t *testing.T) {
	a := resolved(t, "valid-code.json")
	b := resolved(t, "valid-code.json")
	a.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{"x":[]}`)
	b.Item.Parts[0].Config.Samples[0].Expected = json.RawMessage(`{}`)
	if mustHash(t, a) == mustHash(t, b) {
		t.Fatal(`{"x":[]} and {} hashed alike inside a sample`)
	}
}

func TestSeesEdits(t *testing.T) {
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
