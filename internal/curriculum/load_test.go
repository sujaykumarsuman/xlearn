package curriculum

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	seeddata "github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
)

// contentFS copies the embedded curriculum into a MapFS the test can edit.
func contentFS(t *testing.T) fstest.MapFS {
	t.Helper()
	m := fstest.MapFS{}
	err := fs.WalkDir(seeddata.FS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(seeddata.FS, p)
		if err != nil {
			return err
		}
		m[p] = &fstest.MapFile{Data: b, Mode: 0o644}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// editJSON decodes name into v's type, lets fn change it, and writes it back.
func editJSON[T any](t *testing.T, m fstest.MapFS, name string, fn func(*T)) {
	t.Helper()
	f, ok := m[name]
	if !ok {
		t.Fatalf("%s is not in the content FS", name)
	}
	var v T
	if err := json.Unmarshal(f.Data, &v); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	fn(&v)
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	m[name] = &fstest.MapFile{Data: append(b, '\n'), Mode: 0o644}
}

// moveTree renames every path under from/ to to/.
func moveTree(m fstest.MapFS, from, to string) {
	for p, f := range m {
		if rest, ok := strings.CutPrefix(p, from+"/"); ok {
			delete(m, p)
			m[to+"/"+rest] = f
		}
	}
}

// removeTree deletes every path under dir/.
func removeTree(m fstest.MapFS, dir string) {
	for p := range m {
		if strings.HasPrefix(p, dir+"/") {
			delete(m, p)
		}
	}
}

func TestLoadEmbeddedContent(t *testing.T) {
	c, err := LoadContent(seeddata.FS)
	if err != nil {
		t.Fatalf("LoadContent: %v", err)
	}
	if len(c.Courses) != 6 {
		t.Fatalf("courses = %d, want 6 (every paths.json row has a manifest)", len(c.Courses))
	}
	var dsa *CourseContent
	for i := range c.Courses {
		if c.Courses[i].Slug == course.DSASlug {
			dsa = &c.Courses[i]
		}
	}
	if dsa == nil {
		t.Fatal("no dsa course")
	}
	if len(dsa.Items) != 14 || len(dsa.Concepts) != 9 || len(dsa.Weeks) != 16 || len(dsa.Phases) != 4 {
		t.Fatalf("dsa: %d items, %d concepts, %d weeks, %d phases; want 14, 9, 16, 4",
			len(dsa.Items), len(dsa.Concepts), len(dsa.Weeks), len(dsa.Phases))
	}
	if len(c.Lock.Items) != 14 {
		t.Fatalf("ids.lock.json has %d items, want 14", len(c.Lock.Items))
	}
	for _, it := range dsa.Items {
		if le := c.Lock.Items[it.Item.ID]; le.Course != "dsa" || le.Status != it.Item.Status {
			t.Errorf("item %s: lock entry %+v does not match the item", it.Item.ID, le)
		}
		if it.Item.Review != nil && it.Item.Review.Statement != "" {
			t.Errorf("item %s: review.statement is stamped; the m1-09 drafts land unverified", it.Item.ID)
		}
	}
	// Code sections carry their language; prose sections none.
	for _, it := range dsa.Items {
		for _, s := range it.Sections {
			if (s.Kind == "code") != (s.Language == "go") || (s.Kind == "code") != (s.Code != "") || (s.Kind == "code") == (s.BodyMD != "") {
				t.Errorf("item %s: section %+v mixes prose and code", it.Item.ID, s)
			}
		}
	}
	seed, err := c.SeedContent()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range seed.Problems {
		if !strings.HasPrefix(p.ContentHash, "sha256:") {
			t.Errorf("problem %s: content_hash %q", p.ID, p.ContentHash)
		}
	}
}

// The two rewritten Example-1s follow the shape rule and are original drafts.
func TestExampleOneRewrites(t *testing.T) {
	c, err := LoadContent(seeddata.FS)
	if err != nil {
		t.Fatal(err)
	}
	examples := map[string]string{}
	for _, cc := range c.Courses {
		for _, it := range cc.Items {
			for _, s := range it.Sections {
				if s.Kind == "example" {
					examples[it.Item.ID] = s.BodyMD
				}
			}
			if id := it.Item.ID; id == "3" || id == "16" {
				if it.Item.Provenance.Origin != "original" {
					t.Errorf("item %s: provenance.origin %q, want original", id, it.Item.Provenance.Origin)
				}
			}
		}
	}
	for id, body := range map[string]string{
		"3":  "nums = [14,-9,31,-22,5,48,-3], target = -17 → [3,4] because -22 + 5 = -17.",
		"16": "nums = [5, -8, 3, 3, -6, 5, 12, -4] → [[-8, -4, 12], [-8, 3, 5], [-6, 3, 3]].",
	} {
		if !strings.HasPrefix(examples[id], body) {
			t.Errorf("item %s example = %q, want it to start with %q", id, examples[id], body)
		}
	}
}

func TestLoadGuards(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(t *testing.T, m fstest.MapFS)
		want string
	}{
		{"id differs from its directory", func(t *testing.T, m fstest.MapFS) {
			moveTree(m, "courses/dsa/items/2", "courses/dsa/items/4")
			editJSON(t, m, "ids.lock.json", func(l *course.IDsLock) { l.Items["4"] = l.Items["2"] })
		}, `id "2" does not match its directory "4"`},
		{"id not in the lock", func(t *testing.T, m fstest.MapFS) {
			editJSON(t, m, "ids.lock.json", func(l *course.IDsLock) { delete(l.Items, "2") })
		}, `id "2" is not in ids.lock.json`},
		{"lock pins another course", func(t *testing.T, m fstest.MapFS) {
			editJSON(t, m, "ids.lock.json", func(l *course.IDsLock) { l.Items["2"] = course.LockEntry{Course: "sql", Status: "live"} })
		}, `locked to course "sql"`},
		{"item moved to another course", func(t *testing.T, m fstest.MapFS) {
			moveTree(m, "courses/dsa/items/2", "courses/sql/items/2")
		}, "items/2"},
		{"reserved course slug", func(t *testing.T, m fstest.MapFS) {
			addCourse(t, m, "api", "ap")
		}, `course slug "api" is a reserved URL segment`},
		{"malformed course slug", func(t *testing.T, m fstest.MapFS) {
			addCourse(t, m, "Bad_Slug", "bs")
		}, "Bad_Slug"},
		{"unexpected file in an item", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/notes.txt"] = &fstest.MapFile{Data: []byte("x")}
		}, "items/2/notes.txt: unexpected file"},
		{"bad section name", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/sections/attempt/1-summary.md"] = &fstest.MapFile{Data: []byte("x")}
		}, "sections/<stage>/<NN>-<kind>.md"},
		{"unknown stage", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/sections/review/01-summary.md"] = &fstest.MapFile{Data: []byte("x")}
		}, `unknown stage "review"`},
		{"stage and order taken twice", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/sections/attempt/01-why.md"] = &fstest.MapFile{Data: []byte("x")}
		}, "stage attempt order 1 is already taken"},
		{"code collides with prose", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/_code/attempt-01.go.snip"] = &fstest.MapFile{Data: []byte("x")}
		}, "stage attempt order 1 is already taken"},
		{"code fragment saved as .go", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/_code/solution-02.go"] = &fstest.MapFile{Data: []byte("x")}
		}, "_code/<stage>-<NN>.<lang>.snip"},
		{"template for an unknown concept", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/concepts/_code/nope.go.snip"] = &fstest.MapFile{Data: []byte("x")}
		}, `no concept "nope"`},
		{"concept links an unknown week", func(t *testing.T, m fstest.MapFS) {
			editJSON(t, m, "courses/dsa/concepts.json", func(cs *[]course.Concept) { (*cs)[0].Weeks = append((*cs)[0].Weeks, 99) })
		}, "links unknown week 99"},
		{"unknown field in item.json", func(t *testing.T, m fstest.MapFS) {
			editJSON(t, m, "courses/dsa/items/2/item.json", func(v *map[string]any) { (*v)["answer"] = "x" })
		}, `unknown field "answer"`},
		{"unknown field in phases.json", func(t *testing.T, m fstest.MapFS) {
			editJSON(t, m, "courses/dsa/phases.json", func(v *[]map[string]any) { (*v)[0]["path_slug"] = "dsa" })
		}, `unknown field "path_slug"`},
		{"unexpected file in a course", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/answers.json"] = &fstest.MapFile{Data: []byte("{}")}
		}, "courses/dsa/answers.json: unexpected file"},
		{"invalid UTF-8", func(t *testing.T, m fstest.MapFS) {
			m["courses/dsa/items/2/sections/attempt/01-summary.md"] = &fstest.MapFile{Data: []byte{0xff, 0xfe}}
		}, "not valid UTF-8"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := contentFS(t)
			tc.edit(t, m)
			_, err := LoadContent(m)
			if err == nil {
				t.Fatalf("LoadContent accepted the edit; want an error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %q does not contain %q", err, tc.want)
			}
		})
	}
}

func TestLoadIgnoresDotfiles(t *testing.T) {
	m := contentFS(t)
	m["courses/dsa/items/2/.DS_Store"] = &fstest.MapFile{Data: []byte{0}}
	m["courses/dsa/.editorconfig"] = &fstest.MapFile{Data: []byte("x")}
	m["courses/dsa/items/.cache/x"] = &fstest.MapFile{Data: []byte("x")}
	if _, err := LoadContent(m); err != nil {
		t.Fatalf("LoadContent with dotfiles: %v", err)
	}
}

func TestLoadSecondLanguageAtSameOrder(t *testing.T) {
	m := contentFS(t)
	m["courses/dsa/items/1/_code/solution-02.py.snip"] = &fstest.MapFile{Data: []byte("def f(): pass")}
	c, err := LoadContent(m)
	if err != nil {
		t.Fatalf("a second language at the same stage and order must load: %v", err)
	}
	var langs []string
	for _, cc := range c.Courses {
		for _, it := range cc.Items {
			if it.Item.ID == "1" {
				for _, s := range it.Sections {
					if s.Kind == "code" {
						langs = append(langs, s.Language)
					}
				}
			}
		}
	}
	if strings.Join(langs, ",") != "go,py" {
		t.Fatalf("code languages = %v, want [go py] in canonical order", langs)
	}
}

// Reference files `_code/solution.{go,cpp,py}` load as the item's References (not
// sections), move content_hash, and are the only whole-file names under _code/.
func TestLoadReferenceFiles(t *testing.T) {
	m := contentFS(t)
	goRef := "package solution\n\nfunc containsDuplicate(nums []int) bool { return false }\n"
	m["courses/dsa/items/1/_code/solution.go"] = &fstest.MapFile{Data: []byte(goRef)}
	m["courses/dsa/items/1/_code/solution.py"] = &fstest.MapFile{Data: []byte("def f():\n    pass\n")}
	c, err := LoadContent(m)
	if err != nil {
		t.Fatalf("reference files must load: %v", err)
	}
	base, err := LoadContent(seeddata.FS)
	if err != nil {
		t.Fatal(err)
	}
	find := func(c *Content) *course.ResolvedItem {
		for _, cc := range c.Courses {
			for i := range cc.Items {
				if cc.Items[i].Item.ID == "1" {
					return &cc.Items[i]
				}
			}
		}
		t.Fatal("item 1 not loaded")
		return nil
	}
	ri, bi := find(c), find(base)
	if ri.References["go"] != goRef || ri.References["py"] == "" || len(ri.References) != 2 {
		t.Fatalf("References = %v", ri.References)
	}
	if len(ri.Sections) != len(bi.Sections) {
		t.Fatalf("a reference file became a section: %d sections, want %d", len(ri.Sections), len(bi.Sections))
	}
	h1, _ := canon.ContentHash(ri)
	h0, _ := canon.ContentHash(bi)
	if h1 == h0 {
		t.Fatal("adding a reference file did not move content_hash")
	}
	for _, bad := range []string{"_code/solution.java", "_code/reference.go", "_code/solution-1.go"} {
		m := contentFS(t)
		m["courses/dsa/items/1/"+bad] = &fstest.MapFile{Data: []byte("x")}
		if _, err := LoadContent(m); err == nil {
			t.Errorf("%s loaded; want an unexpected-file error", bad)
		}
	}
}

// The content hash is computed over the resolved item and moves with any edit.
func TestSeedContentHashes(t *testing.T) {
	base, err := LoadContent(seeddata.FS)
	if err != nil {
		t.Fatal(err)
	}
	m := contentFS(t)
	m["courses/dsa/items/2/sections/attempt/01-summary.md"] = &fstest.MapFile{Data: []byte("changed")}
	edited, err := LoadContent(m)
	if err != nil {
		t.Fatal(err)
	}
	hashes := func(c *Content) map[string]string {
		out := map[string]string{}
		for _, cc := range c.Courses {
			for i := range cc.Items {
				h, err := canon.ContentHash(&cc.Items[i])
				if err != nil {
					t.Fatal(err)
				}
				out[cc.Items[i].Item.ID] = h
			}
		}
		return out
	}
	hb, he := hashes(base), hashes(edited)
	for id := range hb {
		if (hb[id] != he[id]) != (id == "2") {
			t.Errorf("item %s: hash moved = %v, want %v", id, hb[id] != he[id], id == "2")
		}
	}
}

// addCourse adds a coming-soon course (manifest + catalog row) to the content FS.
func addCourse(t *testing.T, m fstest.MapFS, slug, prefix string) {
	t.Helper()
	m["courses/"+slug+"/course.json"] = &fstest.MapFile{Data: []byte(`{"format": 1, "slug": "` + slug +
		`", "title": "T ` + slug + `", "status": "coming_soon", "id_prefix": "` + prefix +
		`", "public_stats": {"default_visible": true}}`)}
	editJSON(t, m, "paths.json", func(rows *[]course.PathRow) {
		*rows = append(*rows, course.PathRow{Slug: slug, Title: "T " + slug, Status: "coming_soon", SortOrder: 99})
	})
}
