package curriculum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"testing"

	seeddata "github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// TestSeedMatchesV1Snapshot seeds the embedded curriculum into a fresh schema and
// compares every seeded row (v1 fields, uuids stripped) with the committed v1 row
// snapshot. The fixture was written by the v1 loader before the one-shot converter ran
// (m1-09 task 1a), so an equal snapshot proves the conversion changed no row. The only
// intended difference is the two rewritten Example-1 bodies (items 3 and 16), which the
// Example-1 commit updates in the fixture itself. Since m1-03 the four v1 fields whose
// columns M1c drops are derived from the v2 columns (see rowSnapshot).
//
//	go test ./internal/curriculum -run TestSeedMatchesV1Snapshot -update   # rewrite (review the diff)
func TestSeedMatchesV1Snapshot(t *testing.T) {
	pool, dsn := testPool(t)
	freshSchema(t, pool, dsn)

	if err := Seed(context.Background(), store.New(pool), seeddata.FS, discardLogger()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	assertSnapshot(t, pool, snapshotFile)
}

// TestAPIMatchesV1Snapshot serves a fresh seed of the embedded curriculum through the
// production handlers (PgStore, the embedded course registry) and compares every v1
// field of every path, problem, section and concept with the committed v1 row snapshot
// (m1-03). The seed no longer writes the columns M1c drops, so this proves the API still
// serves v1's values for them, derived in Go from role, links and templates.
func TestAPIMatchesV1Snapshot(t *testing.T) {
	pool, dsn := testPool(t)
	freshSchema(t, pool, dsn)
	if err := Seed(context.Background(), store.New(pool), seeddata.FS, discardLogger()); err != nil {
		t.Fatalf("seed: %v", err)
	}
	courses, err := course.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	h := NewService(store.New(pool), courses, discardLogger()).Handler()
	b, err := os.ReadFile(snapshotFile)
	if err != nil {
		t.Fatal(err)
	}
	want := normalize(t, b)

	get := func(path string) ([]byte, map[string]any) {
		t.Helper()
		code, body := raw(t, h, path)
		if code != http.StatusOK {
			t.Fatalf("GET %s: status %d (%s)", path, code, body)
		}
		var v map[string]any
		if err := json.Unmarshal(body, &v); err != nil {
			t.Fatal(err)
		}
		return body, v
	}
	same := func(what string, served, row map[string]any, keys ...string) {
		t.Helper()
		for _, k := range keys {
			if !reflect.DeepEqual(served[k], row[k]) {
				t.Errorf("%s: %s = %#v, v1 %#v", what, k, served[k], row[k])
			}
		}
	}

	// Paths: every catalog row (none retired), v1 fields equal.
	_, body := get("/paths")
	served := map[string]map[string]any{}
	for _, p := range body["paths"].([]any) {
		m := p.(map[string]any)
		served[m["slug"].(string)] = m
	}
	if len(served) != len(want["path"]) {
		t.Errorf("GET /paths lists %d paths, v1 %d", len(served), len(want["path"]))
	}
	for _, row := range want["path"] {
		same("path "+row["slug"].(string), served[row["slug"].(string)], row,
			"slug", "title", "status", "summary", "problem_total", "week_total")
	}

	// Problems, by id and through the path index; sections by (stage, order).
	sections := map[string][]map[string]any{}
	for _, row := range want["problem_section"] {
		id := row["problem_id"].(string)
		sections[id] = append(sections[id], row)
	}
	index := map[string]map[string]any{}
	for _, row := range want["problem"] {
		p := row["path_slug"].(string)
		if _, ok := index[p]; ok {
			continue
		}
		_, body := get("/paths/" + p + "/problems")
		for _, x := range body["problems"].([]any) {
			index[x.(map[string]any)["id"].(string)] = x.(map[string]any)
		}
	}
	problemKeys := []string{"id", "path_slug", "week_n", "title", "difficulty", "pattern", "leetcode_url", "neetcode_url", "is_reinforcement"}
	for _, row := range want["problem"] {
		id := row["id"].(string)
		_, body := get("/problems/" + id)
		same("GET /problems/"+id, body["problem"].(map[string]any), row, problemKeys...)
		same("path index "+id, index[id], row, problemKeys...)

		got := map[string]map[string]any{}
		for _, s := range body["sections"].([]any) {
			m := s.(map[string]any)
			got[fmt.Sprintf("%v/%v", m["stage"], m["order"])] = m
		}
		if len(got) != len(sections[id]) {
			t.Errorf("GET /problems/%s: %d sections, v1 %d", id, len(got), len(sections[id]))
		}
		for _, sr := range sections[id] {
			key := fmt.Sprintf("%v/%v", sr["stage"], sr["order"])
			same("problem "+id+" section "+key, got[key], sr, "stage", "order", "kind", "body_md", "code")
		}
	}

	// Concepts, course-scoped; the DSA alias is byte-identical.
	for _, row := range want["concept"] {
		p, slug := row["path_slug"].(string), row["slug"].(string)
		scoped, body := get("/paths/" + p + "/concepts/" + slug)
		same("concept "+p+"/"+slug, body["concept"].(map[string]any), row,
			"slug", "path_slug", "title", "body_md", "when_to_use_md", "code_template")
		if p == course.DefaultSlug {
			if alias, _ := get("/concepts/" + slug); !bytes.Equal(alias, scoped) {
				t.Errorf("GET /concepts/%s differs from the course-scoped body", slug)
			}
		}
	}
}
