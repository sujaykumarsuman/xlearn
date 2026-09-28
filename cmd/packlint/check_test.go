package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
)

func check(t *testing.T, pub, pack string, extra ...string) (int, string) {
	t.Helper()
	return runPacklint(t, nil, append([]string{"check", "--public", pub, "--pack", pack}, extra...)...)
}

func TestCheckClean(t *testing.T) {
	pub := publicRoot(t)
	pack := packRoot(t, pub)
	if code, out := check(t, pub, pack); code != exitOK || !strings.Contains(out, "0 error(s), 0 warning(s)") {
		t.Fatalf("clean pack: exit %d\n%s", code, out)
	}
	// check is the default command.
	if code, out := runPacklint(t, nil, "--public", pub, "--pack", pack); code != exitOK {
		t.Fatalf("default command: exit %d\n%s", code, out)
	}
}

// The acceptance case: a stale contract hash fails, naming the live and accepted hashes.
func TestCheckStaleContractHash(t *testing.T) {
	pub := publicRoot(t)
	pack := packRoot(t, pub)
	old := liveHash(t, pub, "900")
	// A contract edit in the public item: the harness version.
	p := filepath.Join(pub, "courses/dsa/items/900/item.json")
	b, _ := os.ReadFile(p)
	write(t, p, strings.Replace(string(b), `"func-json@1"`, `"func-json@2"`, 1))
	live := liveHash(t, pub, "900")
	if live == old {
		t.Fatal("a harness bump did not move contract_hash")
	}
	code, out := check(t, pub, pack)
	want := "stale contract hash for 900: live sha256:" + canon.Prefix(live) + ", pack accepts [sha256:" + canon.Prefix(old) + "]"
	if code != exitFail || !strings.Contains(out, want) {
		t.Fatalf("exit %d, want 1 with %q in:\n%s", code, want, out)
	}
	// Listing both hashes (either repo ships first) is clean.
	pf := filepath.Join(pack, "courses/dsa/items/900/pack.json")
	b, _ = os.ReadFile(pf)
	write(t, pf, strings.Replace(string(b), `["`+old+`"]`, `["`+old+`", "`+live+`"]`, 1))
	if code, out := check(t, pub, pack); code != exitOK {
		t.Fatalf("both hashes listed: exit %d\n%s", code, out)
	}
	// A content-only edit (a sample) does not make the pack stale.
	b, _ = os.ReadFile(p)
	write(t, p, strings.Replace(string(b), `"expected": "apple"`, `"expected": "pear"`, 1))
	if code, out := check(t, pub, pack); code != exitOK {
		t.Fatalf("content-only edit: exit %d\n%s", code, out)
	}
}

func TestCheckFiles(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(pack string)
		want   string
	}{
		"missing wrong file": {func(pack string) {
			os.Remove(filepath.Join(pack, "courses/dsa/items/900/submissions/wrong/quadratic.go"))
		}, "submissions/wrong/quadratic.go: declared in pack.json but missing"},
		"undeclared file": {func(pack string) {
			write(t, filepath.Join(pack, "courses/dsa/items/900/notes.txt"), "x")
		}, "notes.txt: undeclared file"},
		"undeclared directory": {func(pack string) {
			write(t, filepath.Join(pack, "courses/dsa/items/900/scratch/a.txt"), "x")
		}, "scratch/: undeclared directory"},
		"dotfile": {func(pack string) {
			write(t, filepath.Join(pack, "courses/dsa/items/900/tests/.DS_Store"), "x")
		}, ".DS_Store: dotfiles are not allowed"},
		"missing key file": {func(pack string) {
			os.Remove(filepath.Join(pack, "courses/dsa/items/901/keys/p-structure.json"))
		}, "keys/p-structure.json: declared in pack.json but missing"},
	} {
		t.Run(name, func(t *testing.T) {
			pub := publicRoot(t)
			pack := packRoot(t, pub)
			tc.mutate(pack)
			if code, out := check(t, pub, pack); code != exitFail || !strings.Contains(out, tc.want) {
				t.Fatalf("exit %d, want 1 with %q in:\n%s", code, tc.want, out)
			}
		})
	}
}

func TestCheckRules(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(t *testing.T, pub, pack string)
		code   int
		want   string
		strict bool
	}{
		"rule 1 unsupported format": {func(t *testing.T, pub, pack string) {
			write(t, filepath.Join(pack, "pack.json"), `{"format_major": 9, "version": "9.0.0"}`)
		}, exitFail, "[rule 1] pack.json: format_major 9 is not supported", false},
		"rule 2 unknown field": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) { m["answers"] = []any{} })
		}, exitFail, `unknown field "answers"`, false},
		"rule 2 item mismatch": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) { m["item"] = "9" })
		}, exitFail, `item "9" does not match its directory "900"`, false},
		"rule 2 no public item": {func(t *testing.T, pub, pack string) {
			write(t, filepath.Join(pack, "courses/dsa/items/950/pack.json"),
				`{"item": "950", "accepts_contract_hashes": ["sha256:`+strings.Repeat("0", 64)+`"]}`)
		}, exitFail, `no public item "950"`, false},
		"rule 2 bad category": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) {
				m["wrong"].([]any)[0].(map[string]any)["category"] = "typo"
			})
		}, exitFail, `category "typo" is not a mistake category`, false},
		"rule 2 not live": {func(t *testing.T, pub, pack string) {
			editItem(t, pub, "900", func(it *course.Item) { it.Status = "retired" })
			lock := filepath.Join(pub, "ids.lock.json")
			b, _ := os.ReadFile(lock)
			write(t, lock, strings.Replace(string(b), `"900": {"course": "dsa", "status": "live"}`, `"900": {"course": "dsa", "status": "retired"}`, 1))
			retarget(t, pub, pack, "900")
		}, exitOK, "packed item is not live (retired)", false},
		"rule 3 three hashes": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) {
				m["accepts_contract_hashes"] = []any{"sha256:" + strings.Repeat("1", 64), "sha256:" + strings.Repeat("2", 64), liveHash(t, pub, "900")}
			})
		}, exitFail, "[rule 3]", false},
		"rule 5 stamped with one wrong": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) { m["wrong"] = m["wrong"].([]any)[:1] })
		}, exitFail, "review.tests is stamped but only 1 wrong solution(s)", false},
		"rule 5 unstamped warns": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) { m["wrong"] = m["wrong"].([]any)[:1]; delete(m, "review") })
		}, exitOK, "WARN [rule 5]", false},
		"rule 5 unstamped strict": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "900", func(m map[string]any) { m["wrong"] = m["wrong"].([]any)[:1]; delete(m, "review") })
		}, exitFail, "WARN [rule 5]", true},
		"rule 6 probe key missing": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "901", func(m map[string]any) { delete(m, "keys") })
			os.RemoveAll(filepath.Join(pack, "courses/dsa/items/901/keys"))
		}, exitFail, `key_source: pack probe "p-structure" has no key file`, false},
		"rule 6 stray key": {func(t *testing.T, pub, pack string) {
			editPack(t, pack, "901", func(m map[string]any) { m["keys"].(map[string]any)["p-pattern"] = "keys/p-structure.json" })
		}, exitFail, `keys["p-pattern"] names no key-graded part`, false},
		"rule 9 unpacked auto item": {func(t *testing.T, pub, pack string) {
			os.RemoveAll(filepath.Join(pack, "courses/dsa/items/900"))
		}, exitOK, "INFO [rule 9] courses/dsa/items/900: has an auto code part and no pack yet", false},
	} {
		t.Run(name, func(t *testing.T) {
			pub := publicRoot(t)
			pack := packRoot(t, pub)
			tc.mutate(t, pub, pack)
			var extra []string
			if tc.strict {
				extra = append(extra, "--strict")
			}
			code, out := check(t, pub, pack, extra...)
			if code != tc.code || !strings.Contains(out, tc.want) {
				t.Fatalf("exit %d (want %d), want %q in:\n%s", code, tc.code, tc.want, out)
			}
		})
	}
}

// Rules 7 and 8 compare the public item with a git ref (--since).
func TestCheckSince(t *testing.T) {
	pub := publicRoot(t)
	git(t, pub, "init", "-q", "-b", "main")
	git(t, pub, "config", "user.email", "packlint-test")
	git(t, pub, "config", "user.name", "test")
	git(t, pub, "config", "commit.gpgsign", "false")
	git(t, pub, "add", "-A")
	git(t, pub, "commit", "-q", "-m", "base")
	pack := packRoot(t, pub)

	editItem(t, pub, "900", func(it *course.Item) { it.Parts[0].Config.Limits.TimeMS = 500 })
	editItem(t, pub, "901", func(it *course.Item) { it.Revision.Probes[2].Config.Options[0].Label = "Sorting" })
	write(t, filepath.Join(pub, "courses/dsa/items/900/sections/attempt/01-summary.md"), "A statement.")
	code, out := check(t, pub, pack, "--since", "HEAD")
	if code != exitOK {
		t.Fatalf("advisory rules must not fail without --strict: exit %d\n%s", code, out)
	}
	for _, want := range []string{
		"WARN [rule 8] courses/dsa/items/900: content-only edits since HEAD (parts.solution.config.limits, sections/attempt (statement)): re-run `make packcheck ITEM=900`",
		"WARN [rule 7]", "label-edit-ok: 901/p-structure/a",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if code, _ := check(t, pub, pack, "--since", "HEAD", "--strict"); code != exitFail {
		t.Fatalf("--strict must fail on the advisory WARNs, got %d", code)
	}
}

func TestCheckJSONAndItemFilter(t *testing.T) {
	pub := publicRoot(t)
	pack := packRoot(t, pub)
	os.Remove(filepath.Join(pack, "courses/dsa/items/901/keys/p-structure.json"))
	code, out := check(t, pub, pack, "--json", "--item", "901")
	var res struct {
		Findings []finding
		Errors   int
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil {
		t.Fatalf("--json output: %v\n%s", err, out)
	}
	if code != exitFail || res.Errors != 1 || res.Findings[0].Item != "901" || res.Findings[0].Rule != 4 {
		t.Fatalf("exit %d, %+v", code, res)
	}
	if code, out := check(t, pub, pack, "--item", "900"); code != exitOK {
		t.Fatalf("--item 900 must ignore 901's problem: %d\n%s", code, out)
	}
}

// hash prints canon's hashes: content over the resolved item, contract over item.json.
func TestHashMatchesCanon(t *testing.T) {
	pub := publicRoot(t)
	code, out := runPacklint(t, nil, "hash", "--public", pub)
	if code != exitOK {
		t.Fatalf("exit %d %s", code, out)
	}
	pc, err := loadPublic(pub)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("hash printed %d lines:\n%s", len(lines), out)
	}
	for i, id := range []string{"900", "901"} {
		ri := pc.items[id].resolved
		content, _ := canon.ContentHash(ri)
		contract, _ := canon.ContractHash(&ri.Item)
		if want := id + " " + content + " " + contract; lines[i] != want {
			t.Errorf("line %d = %q, want %q", i, lines[i], want)
		}
	}
	// The repo root works too (curriculum/courses), and a self-path item prints "-".
	code, out = runPacklint(t, nil, "hash", "--public", "../..", "--item", "1")
	if code != exitOK || !strings.HasSuffix(strings.TrimSpace(out), " -") {
		t.Fatalf("hash --public ../.. --item 1: %d %q", code, out)
	}
	if code, _ := runPacklint(t, nil, "hash", "--public", pub, "--item", "nope"); code != exitUsage {
		t.Fatalf("unknown item: exit %d, want 2", code)
	}
}

func TestUsage(t *testing.T) {
	for _, args := range [][]string{{"bogus"}, {"check", "--public", t.TempDir()}, {"fingerprint", "--pack", t.TempDir()}} {
		if code, _ := runPacklint(t, nil, args...); code != exitUsage {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
}

// --- helpers ---

func editPack(t *testing.T, pack, id string, fn func(map[string]any)) {
	t.Helper()
	p := filepath.Join(pack, "courses/dsa/items", id, "pack.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	fn(m)
	out, _ := json.MarshalIndent(m, "", "  ")
	write(t, p, string(out))
}

func editItem(t *testing.T, pub, id string, fn func(*course.Item)) {
	t.Helper()
	p := filepath.Join(pub, "courses/dsa/items", id, "item.json")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	it, err := course.DecodeItem(b)
	if err != nil {
		t.Fatal(err)
	}
	fn(it)
	out, _ := json.MarshalIndent(it, "", "  ")
	write(t, p, string(out))
}

// retarget points an item's pack at its current live contract hash.
func retarget(t *testing.T, pub, pack, id string) {
	t.Helper()
	h := liveHash(t, pub, id)
	editPack(t, pack, id, func(m map[string]any) { m["accepts_contract_hashes"] = []any{h} })
}
