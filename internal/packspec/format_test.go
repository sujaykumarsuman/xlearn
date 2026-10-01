package packspec

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// Every value here is synthetic and hand-made (never derived from xlearn-evalpack).

func testSig(t *testing.T) *harness.Sig {
	t.Helper()
	s, err := harness.ParseSig(harness.FuncJSON, &course.Signature{Mode: "function", Name: "f",
		Params: []course.Param{{Name: "nums", Type: "int[]"}, {Name: "word", Type: "string"}}, Returns: "int"})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCaseLineAndID(t *testing.T) {
	in, err := harness.CanonicalInput(testSig(t), []byte(`{"args": [[3, 1], "ab"]}`))
	if err != nil {
		t.Fatal(err)
	}
	c := &Case{ID: CaseID(in), Input: in, Expected: []byte("4"), Tags: []string{"edge", "small"}}
	line, err := c.Line()
	if err != nil {
		t.Fatal(err)
	}
	want := `{"args":[[3,1],"ab"],"expected":4,"id":"` + c.ID + `","tags":["edge","small"]}`
	if string(line) != want {
		t.Fatalf("line\n got %s\nwant %s", line, want)
	}
	// The id hashes the input only (a fixed vector).
	if c.ID != CaseID([]byte(`{"args":[[3,1],"ab"]}`)) || !strings.HasPrefix(c.ID, "sha256:") || len(c.ID) != 71 {
		t.Fatalf("id %s", c.ID)
	}
	back, err := ParseLine(line)
	if err != nil || back.ID != c.ID || !bytes.Equal(back.Input, in) || back.Group() != GroupEdge {
		t.Fatalf("ParseLine: %+v %v", back, err)
	}
	// Non-canonical or tampered lines are rejected.
	for _, bad := range []string{
		`{"args":[[3,1],"ab"], "expected":4,"id":"` + c.ID + `","tags":["edge","small"]}`,
		`{"args":[[3,2],"ab"],"expected":4,"id":"` + c.ID + `","tags":["edge","small"]}`,
		`{"args":[[3,1],"ab"],"expected":4,"extra":1,"id":"` + c.ID + `","tags":["edge","small"]}`,
	} {
		if _, err := ParseLine([]byte(bad)); err == nil {
			t.Errorf("ParseLine accepted %s", bad)
		}
	}

	perf := &Case{Gen: "int_array@1", Params: []byte(`{"max":9,"min":0,"n":5}`), Seed: 18446744073709551615, Expected: []byte(`{"bytes":10,"sha256":"00"}`), Tags: []string{"perf"}}
	perf.ID = CaseID(perf.CanonicalInput())
	if string(perf.CanonicalInput()) != `{"gen":"int_array@1","params":{"max":9,"min":0,"n":5},"seed":18446744073709551615}` {
		t.Fatalf("perf input %s", perf.CanonicalInput())
	}
	pl, err := perf.Line()
	if err != nil {
		t.Fatal(err)
	}
	if back, err := ParseLine(pl); err != nil || back.Seed != perf.Seed || back.Group() != GroupPerf {
		t.Fatalf("perf ParseLine: %+v %v", back, err)
	}
}

func TestSortAndEncodeCases(t *testing.T) {
	mk := func(input, group string) *Case {
		in := []byte(`{"args":[[` + input + `],"x"]}`)
		return &Case{ID: CaseID(in), Input: in, Expected: []byte("0"), Tags: []string{group}}
	}
	cs := []*Case{mk("1", GroupPerf), mk("2", GroupRandom), mk("3", GroupEdge), mk("4", GroupRandom), mk("5", GroupEdge)}
	SortCases(cs)
	var order []string
	for _, c := range cs {
		order = append(order, c.Group())
	}
	if got := strings.Join(order, ","); got != "edge,edge,random,random,perf" {
		t.Fatalf("order %s", got)
	}
	if !strings.Contains(string(cs[0].Input), "3") || !strings.Contains(string(cs[2].Input), "2") {
		t.Fatal("SortCases is not stable within a group")
	}
	a, err := EncodeCases(cs)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := EncodeCases(cs)
	if !bytes.Equal(a, b) {
		t.Fatal("EncodeCases is not deterministic")
	}
	back, err := DecodeCases(a)
	if err != nil || len(back) != len(cs) {
		t.Fatalf("DecodeCases: %d %v", len(back), err)
	}
	empty, _ := EncodeCases(nil)
	if len(empty) == 0 {
		t.Fatal("an empty case set must still be a zstd frame")
	}
	if raw, err := Decompress(empty); err != nil || len(raw) != 0 {
		t.Fatalf("empty frame: %q %v", raw, err)
	}
}

func TestLockRoundTripAndSeed(t *testing.T) {
	l := NewLock(CurrentTools(map[string]string{"go": "golang@sha256:aa"}))
	id, h := "sha256:"+strings.Repeat("a", 64), "sha256:"+strings.Repeat("b", 64)
	l.Cases["fixture/fx-001"] = map[string]string{id: h}
	b, err := l.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := DecodeLock(b)
	if err != nil || !back.Tools.Equal(l.Tools) || back.Cases["fixture/fx-001"][id] != h {
		t.Fatalf("DecodeLock: %+v %v", back, err)
	}
	again, _ := back.Encode()
	if !bytes.Equal(again, b) {
		t.Fatal("lock encoding is not byte-stable")
	}
	if l.Tools.Packlint != LockVersion || !strings.Contains(l.Tools.Harness, "func-json@1") || l.Tools.Gen == "" {
		t.Fatalf("tools %+v", l.Tools)
	}
	if s, err := DecodeLock([]byte("{}\n")); err != nil || len(s.Cases) != 0 {
		t.Fatalf("scaffold lock: %v", err)
	}
	for _, bad := range []string{`{"format":2,"cases":{},"tools":{}}`, `{"format":1,"cases":{"x":{"id":"h"}},"tools":{}}`, `{"format":1,"cases":{},"tools":{},"x":1}`} {
		if _, err := DecodeLock([]byte(bad)); err == nil {
			t.Errorf("DecodeLock accepted %s", bad)
		}
	}
	// Seed: a fixed vector (changing the rule changes every generated case: LockVersion).
	if got := Seed("fx-001", "gen/random.go", `["--n","5"]`, 3); got != Seed("fx-001", "gen/random.go", `["--n","5"]`, 3) || got == Seed("fx-001", "gen/random.go", `["--n","5"]`, 4) {
		t.Fatal("Seed is not deterministic per index")
	}
	if Seed("fx-001", "a", "", 0) == Seed("fx-00", "1a", "", 0) {
		t.Fatal("Seed fields are not separated")
	}
	d := Diff(map[string]string{"a": "1", "b": "2"}, map[string]string{"b": "3", "c": "4"})
	if d.Missing != 1 || d.Changed != 1 || d.Extra != 1 || d.Empty() {
		t.Fatalf("Diff %+v", d)
	}
}

func TestGenericValidator(t *testing.T) {
	sig := testSig(t)
	v, err := NewValidator(sig, []course.Constraint{
		{Arg: "nums", Len: []int{1, 3}}, {Arg: "nums[*]", Range: []float64{-5, 5}}, {Arg: "word", Len: []int{0, 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	check := func(raw string) error {
		in, err := harness.DecodeInput(sig, []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		return v.Check(in)
	}
	if err := check(`{"args":[[1,-5,5],"éé"]}`); err != nil {
		t.Fatalf("valid input: %v", err)
	}
	for _, bad := range []string{`{"args":[[],"a"]}`, `{"args":[[1,2,3,4],"a"]}`, `{"args":[[6],"a"]}`, `{"args":[[1],"abc"]}`} {
		if err := check(bad); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: %v, want ErrInvalidInput", bad, err)
		}
	}
	// Constraints the grammar cannot apply.
	for _, c := range []course.Constraint{
		{Arg: "nope", Len: []int{0, 1}}, {Arg: "nums", Range: []float64{0, 1}}, {Arg: "word[*]", Len: []int{0, 1}},
		{Arg: "nums[*][*]", Range: []float64{0, 1}}, {Arg: "nums.len", Len: []int{0, 1}}, {Arg: "nums"},
	} {
		if _, err := NewValidator(sig, []course.Constraint{c}); !errors.Is(err, ErrConstraint) {
			t.Errorf("%+v: %v, want ErrConstraint", c, err)
		}
	}
	// Class mode: a name matches the constructor's and every op's params; trees count nodes.
	cls, err := harness.ParseSig(harness.ClassOps, &course.Signature{Mode: "class", Name: "C",
		Params: []course.Param{{Name: "k", Type: "int"}},
		Ops:    []course.Op{{Name: "Add", Params: []course.Param{{Name: "k", Type: "int"}, {Name: "t", Type: "TreeNode"}}}}})
	if err != nil {
		t.Fatal(err)
	}
	cv, err := NewValidator(cls, []course.Constraint{{Arg: "k", Range: []float64{0, 9}}, {Arg: "t", Len: []int{0, 2}}, {Arg: "t[*]", Range: []float64{0, 9}}})
	if err != nil {
		t.Fatal(err)
	}
	ok, _ := harness.DecodeInput(cls, []byte(`{"ops":["C","Add"],"args":[[1],[2,[1,null,2]]]}`))
	if err := cv.Check(ok); err != nil {
		t.Fatalf("class valid: %v", err)
	}
	for _, raw := range []string{`{"ops":["C","Add"],"args":[[10],[2,[]]]}`, `{"ops":["C","Add"],"args":[[1],[12,[]]]}`,
		`{"ops":["C","Add"],"args":[[1],[2,[1,2,3]]]}`, `{"ops":["C","Add"],"args":[[1],[2,[1,null,10]]]}`} {
		in, err := harness.DecodeInput(cls, []byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if err := cv.Check(in); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: %v", raw, err)
		}
	}
}

func testManifest() (*Manifest, fstest.MapFS) {
	data := []byte("synthetic cases")
	key := []byte(`{"q":"synthetic"}`)
	files := map[string]string{
		"courses/zz/items/zz-001/cases.jsonl.zst": FileHash(data),
		"courses/zz/items/zz-001/keys/q.json":     FileHash(key),
	}
	m := &Manifest{FormatMajor: 1, Version: "1.2.3", ValidatedAgainst: strings.Repeat("ab", 20), Items: map[string]ManifestItem{
		"zz-001": {ItemHash: ItemHash(files), AcceptsContractHashes: []string{"sha256:" + strings.Repeat("c", 64)}, GraderKinds: []string{"code", "key"}, Files: files},
	}}
	fsys := fstest.MapFS{
		"courses/zz/items/zz-001/cases.jsonl.zst": {Data: data},
		"courses/zz/items/zz-001/keys/q.json":     {Data: key},
	}
	return m, fsys
}

func TestManifestReadVerify(t *testing.T) {
	m, fsys := testManifest()
	b, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	back, err := ReadManifest(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := back.Encode(); !bytes.Equal(again, b) {
		t.Fatal("manifest encoding is not byte-stable")
	}
	if bad := back.VerifyFiles(fsys); len(bad) != 0 {
		t.Fatalf("VerifyFiles: %v", bad)
	}
	fsys["courses/zz/items/zz-001/keys/q.json"] = &fstest.MapFile{Data: []byte("tampered")}
	if bad := back.VerifyFiles(fsys); bad["zz-001"] == nil {
		t.Fatal("a tampered file verified")
	}
	delete(fsys, "courses/zz/items/zz-001/cases.jsonl.zst")
	if bad := back.VerifyFiles(fsys); bad["zz-001"] == nil {
		t.Fatal("a missing file verified")
	}
	// Strict: exactly t1 §3.3's fields; hashes and paths checked.
	for name, mutate := range map[string]func(string) string{
		"extra field": func(s string) string { return strings.Replace(s, `"version"`, `"course": "zz", "version"`, 1) },
		"format":      func(s string) string { return strings.Replace(s, `"format_major": 1`, `"format_major": 2`, 1) },
		"item hash": func(s string) string {
			return strings.Replace(s, back.Items["zz-001"].ItemHash, "sha256:"+strings.Repeat("0", 64), 1)
		},
		"sha":            func(s string) string { return strings.Replace(s, strings.Repeat("ab", 20), "main", 1) },
		"trailing":       func(s string) string { return s + "{}" },
		"grader kind":    func(s string) string { return strings.Replace(s, `"key"`, `"oracle"`, 1) },
		"disallowed dir": func(s string) string { return strings.ReplaceAll(s, "keys/q.json", "gen/q.go") },
	} {
		if _, err := ReadManifest(strings.NewReader(mutate(string(b)))); err == nil {
			t.Errorf("%s: ReadManifest accepted it", name)
		}
	}
}

func TestBuildAndListing(t *testing.T) {
	dir := t.TempDir()
	m, fsys := testManifest()
	for p, f := range fsys {
		if err := writeFile(dir, p, f.Data); err != nil {
			t.Fatal(err)
		}
	}
	mb, _ := m.Encode()
	if err := writeFile(dir, ManifestFile, mb); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir, DockerfileName, Dockerfile([]string{"zz"})); err != nil {
		t.Fatal(err)
	}
	l, err := ListDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if errs := CheckListing(l); len(errs) != 0 {
		t.Fatalf("clean build dir: %v", errs)
	}
	// A planted generator, a source file and a dotfile all fail.
	for _, planted := range []string{"courses/zz/items/zz-001/gen/x.go", "courses/zz/items/zz-001/pack.json", "courses/zz/items/zz-001/keys/.hidden", "tests.lock"} {
		d2 := t.TempDir()
		if err := os.CopyFS(d2, os.DirFS(dir)); err != nil {
			t.Fatal(err)
		}
		if err := writeFile(d2, planted, []byte("x")); err != nil {
			t.Fatal(err)
		}
		l2, err := ListDir(d2)
		if err != nil {
			t.Fatal(err)
		}
		if errs := CheckListing(l2); len(errs) == 0 {
			t.Errorf("planted %s passed the listing test", planted)
		}
	}
	// Two courses in one layer, and a manifest that is not last.
	bad := &Listing{Layers: [][]string{{"courses/a/items/a-001/cases.jsonl.zst", "courses/b/items/b-001/cases.jsonl.zst"}, {ManifestFile}},
		Read: func(string) ([]byte, error) { return nil, os.ErrNotExist }}
	if errs := CheckListing(bad); len(errs) == 0 {
		t.Error("two courses in one layer passed")
	}
	if !strings.Contains(string(Dockerfile([]string{"b", "a"})), "COPY courses/a/ /courses/a/\nCOPY courses/b/ /courses/b/\nCOPY manifest.json /manifest.json\n") {
		t.Errorf("Dockerfile:\n%s", Dockerfile([]string{"b", "a"}))
	}
	if _, err := os.Stat(filepath.Join(dir, ManifestFile)); err != nil {
		t.Fatal(err)
	}
}

func TestItemFormat1(t *testing.T) {
	h := "sha256:" + strings.Repeat("a1", 32)
	good := `{"item": "zz-001", "accepts_contract_hashes": ["` + h + `"],
	  "gen": [{"cmd": "gen/random.go", "args": ["--n", "5"], "count": 3, "tags": ["random"]},
	          {"spec": {"gen": "int_array@1", "params": {"n": 10, "min": 0, "max": 9}}, "count": 2, "tags": ["perf"]}],
	  "large_case_exception": {"max_bytes": 1048576, "stamped": "2026-10-01"}}`
	it, err := DecodeItem([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if err := it.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if it.LiteralLimit() != 1<<20 || !strings.Contains(strings.Join(it.Files(), ","), "gen/random.go") {
		t.Fatalf("limit %d files %v", it.LiteralLimit(), it.Files())
	}
	for name, tc := range map[string]struct {
		g    []GenEntry
		want string
	}{
		"both":         {[]GenEntry{{Cmd: "gen/a.go", Spec: nil, Count: 1}, {Count: 1}}, "exactly one of cmd or spec"},
		"not go":       {[]GenEntry{{Cmd: "gen/a.py", Count: 1}}, "Go programs"},
		"outside gen":  {[]GenEntry{{Cmd: "tests/a.go", Count: 1}}, "under gen/"},
		"count":        {[]GenEntry{{Cmd: "gen/a.go", Count: 0}}, "count"},
		"edge tag":     {[]GenEntry{{Cmd: "gen/a.go", Count: 1, Tags: []string{"edge"}}}, "edge cases live"},
		"bad tag":      {[]GenEntry{{Cmd: "gen/a.go", Count: 1, Tags: []string{"Big"}}}, "must match"},
		"unknown spec": {[]GenEntry{{Spec: specOf("nope@1"), Count: 1}}, "spec.gen"},
	} {
		x := Item{Item: "zz-001", AcceptsContractHashes: []string{h}, Gen: tc.g}
		if err := x.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", name, err, tc.want)
		}
	}
	x := Item{Item: "zz-001", AcceptsContractHashes: []string{h}, LargeCaseException: &LargeCaseException{MaxBytes: 4 << 20}}
	if err := x.Validate(); err == nil {
		t.Error("a 4 MiB exception validated")
	}
	if x := (Item{LargeCaseException: &LargeCaseException{MaxBytes: 1 << 20}}); x.LiteralLimit() != LiteralCap {
		t.Error("an unstamped exception raised the cap")
	}
}
