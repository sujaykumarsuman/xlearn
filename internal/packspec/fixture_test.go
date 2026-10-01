package packspec_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/packspec"
	"github.com/sujaykumarsuman/xlearn/internal/packspec/executor"
	"github.com/sujaykumarsuman/xlearn/internal/packspec/gen"
	"github.com/sujaykumarsuman/xlearn/internal/platform/harness"
)

// The committed SYNTHETIC fixture pack (internal/judge/testdata/pack) is internally
// consistent and deterministic — checked with plain `go test`, no docker: every line of
// every cases.jsonl.zst is canonical, hashes to its tests.lock entry and decodes against
// the public signature; re-compressing the decoded lines gives the same bytes; manifest.json
// (file hashes, item_hash) recomputes identically and passes the listing test. The source →
// pack rebuild runs the Go reference, so it needs docker: the pack-fixture CI job, or
// `go test ./internal/packspec -run TestFixturePack -update` to rewrite pack/ locally.

var update = flag.Bool("update", false, "rebuild internal/judge/testdata/pack from packsrc (needs docker)")

const (
	content = "../judge/testdata/content"
	packsrc = "../judge/testdata/packsrc"
	pack    = "../judge/testdata/pack"
	// fixtureSHA is the fixture's validated_against: it is validated against its own tree.
	fixtureSHA = "0000000000000000000000000000000000000000"
)

func TestFixturePack(t *testing.T) {
	c, err := curriculum.LoadContent(os.DirFS(content))
	if err != nil {
		t.Fatalf("the fixture content root must pass the loader and its guards: %v", err)
	}
	if *update {
		rebuild(t, c)
	}
	rb, err := os.ReadFile(filepath.Join(packsrc, packspec.RootFile))
	if err != nil {
		t.Fatal(err)
	}
	root, err := packspec.DecodeRoot(rb)
	if err != nil || root.FormatMajor != packspec.FormatMajor {
		t.Fatalf("packsrc root: %+v %v", root, err)
	}
	lb, err := os.ReadFile(filepath.Join(packsrc, packspec.LockFile))
	if err != nil {
		t.Fatal(err)
	}
	lock, err := packspec.DecodeLock(lb)
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := lock.Encode(); !bytes.Equal(again, lb) {
		t.Error("tests.lock is not in its canonical encoding")
	}

	mb, err := os.ReadFile(filepath.Join(pack, packspec.ManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	m, err := packspec.ReadManifest(bytes.NewReader(mb))
	if err != nil {
		t.Fatal(err)
	}
	if again, _ := m.Encode(); !bytes.Equal(again, mb) {
		t.Error("manifest.json does not re-encode identically")
	}
	if m.Version != root.Version || m.ValidatedAgainst != fixtureSHA {
		t.Errorf("manifest identity %s %s", m.Version, m.ValidatedAgainst)
	}
	if bad := m.VerifyFiles(os.DirFS(pack)); len(bad) != 0 {
		t.Fatalf("VerifyFiles: %v", bad)
	}
	l, err := packspec.ListDir(pack)
	if err != nil {
		t.Fatal(err)
	}
	if errs := packspec.CheckListing(l); len(errs) != 0 {
		t.Fatalf("listing test: %v", errs)
	}
	if len(m.Items) != 3 {
		t.Fatalf("want the 3 fixture items, got %d", len(m.Items))
	}

	pub := map[string]int{}
	for i, ri := range c.Courses[0].Items {
		pub[ri.Item.ID] = i
	}
	for id, it := range m.Items {
		ri := &c.Courses[0].Items[pub[id]]
		live, err := canon.ContractHash(&ri.Item)
		if err != nil || !slices.Contains(it.AcceptsContractHashes, live) {
			t.Errorf("%s: the manifest does not accept the live contract hash", id)
		}
		if it.ItemHash != packspec.ItemHash(it.Files) {
			t.Errorf("%s: item_hash", id)
		}
		part := ri.Item.Parts[0].Config
		sig, err := harness.ParseSig(part.Harness, part.Signature)
		if err != nil {
			t.Fatal(err)
		}
		file := path.Join("courses", it.ItemCourse(), "items", id, packspec.CasesFile)
		zb, err := os.ReadFile(filepath.Join(pack, filepath.FromSlash(file)))
		if err != nil {
			t.Fatal(err)
		}
		raw, err := packspec.Decompress(zb)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(packspec.Compress(raw), zb) {
			t.Errorf("%s: re-compressing the decoded lines does not give the same bytes", id)
		}
		locked := lock.Cases[packspec.ItemKey(it.ItemCourse(), id)]
		seen := map[string]bool{}
		rank := map[string]int{packspec.GroupEdge: 0, packspec.GroupRandom: 1, packspec.GroupPerf: 2}
		last := 0
		err = packspec.ScanLines(bytes.NewReader(raw), func(line []byte) error {
			cs, err := packspec.ParseLine(line)
			if err != nil {
				t.Errorf("%s: %v", id, err)
				return nil
			}
			seen[cs.ID] = true
			if locked[cs.ID] != packspec.LineHash(line) {
				t.Errorf("%s: case %s does not hash to its tests.lock entry", id, cs.ID)
			}
			if r := rank[cs.Group()]; r < last {
				t.Errorf("%s: case %s is out of group order", id, cs.ID)
			} else {
				last = r
			}
			var in *harness.Input
			if cs.IsPerfSpec() {
				var buf bytes.Buffer
				if err := writePerf(&buf, sig, cs); err != nil {
					t.Errorf("%s: perf case %s: %v", id, cs.ID, err)
				}
				in, err = harness.DecodeInput(sig, buf.Bytes())
			} else {
				canonical, cerr := harness.CanonicalInput(sig, cs.Input)
				if cerr != nil || !bytes.Equal(canonical, cs.Input) {
					t.Errorf("%s: case %s input is not canonical for the signature: %v", id, cs.ID, cerr)
				}
				in, err = harness.DecodeInput(sig, cs.Input)
			}
			if err != nil {
				t.Errorf("%s: case %s: %v", id, cs.ID, err)
				return nil
			}
			var digest map[string]json.RawMessage
			if json.Unmarshal(cs.Expected, &digest) == nil && digest["sha256"] != nil {
				return nil
			}
			if _, err := harness.DecodeExpected(sig, in, cs.Expected); err != nil {
				t.Errorf("%s: case %s expected: %v", id, cs.ID, err)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(seen) != len(locked) {
			t.Errorf("%s: %d cases in the pack, %d in tests.lock", id, len(seen), len(locked))
		}
	}
}

// rebuild materializes packsrc with the docker executor and rewrites pack/.
func rebuild(t *testing.T, c *curriculum.Content) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("-update needs docker")
	}
	ex := &executor.Docker{}
	var items []packspec.BuiltItem
	for _, ri := range c.Courses[0].Items {
		s, err := packspec.LoadItemSource(packsrc, "fixture", ri.Item.ID, &ri)
		if err != nil {
			t.Fatal(err)
		}
		m, err := s.Materialize(context.Background(), ex)
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, packspec.BuiltItem{Materialized: m, GraderKinds: []string{"code"}})
	}
	tmp := filepath.Join(t.TempDir(), "pack")
	if _, err := packspec.Build(items, packspec.BuildOptions{Out: tmp, Version: "0.1.0", ValidatedAgainst: fixtureSHA}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(pack); err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(pack, os.DirFS(tmp)); err != nil {
		t.Fatal(err)
	}
	t.Logf("rewrote %s (run packlint lock --write over packsrc if the cases changed)", strings.TrimPrefix(pack, "../"))
}

func writePerf(w io.Writer, sig *harness.Sig, c *packspec.Case) error {
	return gen.WriteInput(w, sig, gen.Spec{Gen: c.Gen, Params: c.Params}, c.Seed)
}
