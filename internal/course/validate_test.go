package course_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// dsaJSON returns the embedded DSA manifest decoded into a generic map, for mutation.
func dsaJSON(t *testing.T) map[string]any {
	t.Helper()
	return readJSONFile(t, curriculum.FS, "courses/dsa/course.json")
}

func dirFS(dir string) fs.FS { return os.DirFS(dir) }

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func at(m map[string]any, path ...string) map[string]any {
	for _, p := range path {
		m = m[p].(map[string]any)
	}
	return m
}

func TestManifestValidateRules(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(m map[string]any)
		want   string
	}{
		{"format", func(m map[string]any) { m["format"] = 2 }, "format: must be 1"},
		{"slug shape", func(m map[string]any) { m["slug"] = "DSA_x" }, "slug:"},
		{"status enum", func(m map[string]any) { m["status"] = "live" }, "status:"},
		{"id_prefix shape", func(m map[string]any) { m["id_prefix"] = "d" }, "id_prefix:"},
		{"active needs the method", func(m map[string]any) { delete(m, "revision") }, "revision: is required for a active course"},
		{"nav screen closed set", func(m map[string]any) {
			at(m, "nav")["groups"].([]any)[0].(map[string]any)["items"].([]any)[0].(map[string]any)["screen"] = "arena"
		}, `"arena" is not one of`},
		{"nav duplicate screen", func(m map[string]any) {
			at(m, "nav")["groups"].([]any)[1].(map[string]any)["items"].([]any)[0].(map[string]any)["screen"] = "today"
		}, `duplicate screen "today"`},
		{"stage duration > 0", func(m map[string]any) { at(m, "stages", "attempt")["duration_s"] = 0 }, "stages.attempt.duration_s: must be > 0"},
		{"reimplement enum", func(m map[string]any) { at(m, "stages")["reimplement"] = "sometimes" }, "stages.reimplement"},
		{"strategy closed set", func(m map[string]any) { at(m, "grading")["strategy"] = "elo@1" }, `"elo@1" is not one of`},
		{"selected strategy needs params", func(m map[string]any) {
			at(m, "grading")["strategy"] = "rubric_pct@1"
		}, `missing params for the selected strategy "rubric_pct@1"`},
		{"verdict_timer free class", func(m map[string]any) {
			at(m, "grading", "params", "verdict_timer@1")["free_classes"] = []any{"CE", "OOPS"}
		}, `"OOPS" is not one of`},
		{"grades are universal", func(m map[string]any) {
			g := m["grades"].([]any)
			g[0], g[1] = g[1], g[0]
		}, "must equal the universal grades"},
		{"ladder is universal", func(m map[string]any) { at(m, "revision")["ladder_days"] = []any{1, 2, 4, 8, 16} }, "universal ladder"},
		{"every level in exactly one band", func(m map[string]any) {
			at(m, "revision")["bands"].([]any)[1].(map[string]any)["levels"] = []any{4}
		}, "ladder level 5 must be covered by exactly one band"},
		{"criterion key is an identifier", func(m map[string]any) {
			at(m, "revision")["bands"].([]any)[0].(map[string]any)["criteria"].([]any)[0].(map[string]any)["key"] = "O(n log n)"
		}, "a criterion identifier, never a value"},
		{"band format needs est_minutes", func(m map[string]any) { delete(at(m, "plan", "est_minutes"), "touch_resolve") }, `missing "touch_resolve"`},
		{"course_attempt minutes", func(m map[string]any) { delete(at(m, "plan", "est_minutes"), "course_attempt") }, `missing "course_attempt"`},
		{"mock minutes when a mock exists", func(m map[string]any) { delete(at(m, "plan", "est_minutes"), "mock") }, `missing "mock"`},
		{"minutes > 0", func(m map[string]any) { at(m, "plan", "est_minutes")["touch_recall"] = 0 }, "must be > 0"},
		{"minutes key shape", func(m map[string]any) { at(m, "plan", "est_minutes")["lunch"] = 30 }, `key "lunch"`},
		{"core categories", func(m map[string]any) {
			cats := at(m, "mistakes")["categories"].([]any)
			at(m, "mistakes")["categories"] = cats[:6] // drops communication, time_management
		}, `missing core category "communication"`},
		{"category id shape", func(m map[string]any) {
			at(m, "mistakes")["categories"].([]any)[1].(map[string]any)["id"] = "Wrong-Pattern"
		}, "mistakes.categories[1].id"},
		{"prefill names a declared category", func(m map[string]any) {
			at(m, "mistakes")["prefill"] = []any{map[string]any{"signal": "tle_perf_only", "category": "slow", "strength": "strong"}}
		}, `"slow" is not a declared category`},
		{"rubric 1..10 dims", func(m map[string]any) { at(m, "mock", "rubric")["dims"] = []any{} }, "must have 1..10 dimensions"},
		{"rubric scale", func(m map[string]any) { at(m, "mock", "rubric")["scale"] = []any{0, 10} }, "must be [1, 5]"},
		{"rail tiles the duration", func(m map[string]any) { at(m, "mock")["duration_s"] = 3600 }, "the last phase must end at duration_s"},
		{"target within max", func(m map[string]any) { at(m, "mock", "targets")["pre"] = 36 }, "mock.targets.pre"},
		{"evidence hint dim", func(m map[string]any) {
			at(m, "mock")["evidence_hints"] = []any{map[string]any{"signal": "tle_perf_only", "dim": "speed", "max": 2}}
		}, `"speed" is not a rubric dimension`},
		{"persona ≤ 600 chars", func(m map[string]any) { at(m, "coach")["persona"] = strings.Repeat("x", 601) }, "max 600"},
		{"coach off_during", func(m map[string]any) { at(m, "coach")["off_during"] = []any{"course"} }, `"course" is not one of`},
		{"metrics closed set (D31: no mock best)", func(m map[string]any) {
			at(m, "public_stats")["metrics"] = []any{"solved", "mock_best"}
		}, `"mock_best" is not one of`},
		{"default_visible required", func(m map[string]any) { delete(at(m, "public_stats"), "default_visible") }, "default_visible: is required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := dsaJSON(t)
			tc.mutate(doc)
			m, err := course.DecodeManifest(mustMarshal(t, doc))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			err = m.Validate()
			if err == nil {
				t.Fatalf("accepted; want an error mentioning %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error does not mention %q:\n%v", tc.want, err)
			}
		})
	}
}

// loadFS builds a curriculum-like FS from the real embedded files, with overrides.
func loadFS(t *testing.T, override map[string]string) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{}
	names, _ := fs.Glob(curriculum.FS, course.ManifestGlob)
	for _, n := range append(names, course.PathsFile) {
		b, err := fs.ReadFile(curriculum.FS, n)
		if err != nil {
			t.Fatal(err)
		}
		fsys[n] = &fstest.MapFile{Data: b}
	}
	for n, data := range override {
		if data == "" {
			delete(fsys, n)
			continue
		}
		fsys[n] = &fstest.MapFile{Data: []byte(data)}
	}
	return fsys
}

func TestLoadStrictness(t *testing.T) {
	sql := `{"format": 1, "slug": "sql", "title": "SQL & Data Modeling", "status": "coming_soon", "id_prefix": "sql", "public_stats": {"default_visible": true}}`
	for _, tc := range []struct {
		name     string
		override map[string]string
		want     string
	}{
		{"real curriculum loads", nil, ""},
		{"unknown field", map[string]string{"courses/sql/course.json": strings.Replace(sql, `"format": 1`, `"format": 1, "colour": "teal"`, 1)}, `unknown field "colour"`},
		{"trailing data", map[string]string{"courses/sql/course.json": sql + `{"format": 1}`}, "trailing data"},
		{"directory must equal slug", map[string]string{"courses/sql/course.json": strings.Replace(sql, `"slug": "sql"`, `"slug": "sequel"`, 1)}, "does not match its directory"},
		{"id_prefix unique", map[string]string{"courses/sql/course.json": strings.Replace(sql, `"id_prefix": "sql"`, `"id_prefix": "sd"`, 1)}, `id_prefix "sd" is used by both`},
		{"paths.json title agrees", map[string]string{"courses/sql/course.json": strings.Replace(sql, "SQL & Data Modeling", "SQL", 1)}, "title"},
		{"paths.json status agrees", map[string]string{"courses/sql/course.json": strings.Replace(sql, "coming_soon", "retired", 1)}, "status"},
		{"every paths.json row has a manifest", map[string]string{"courses/sql/course.json": ""}, `row "sql" has no manifest`},
		{"every manifest has a paths.json row", map[string]string{"courses/extra/course.json": strings.NewReplacer(`"slug": "sql"`, `"slug": "extra"`, `"id_prefix": "sql"`, `"id_prefix": "ext"`).Replace(sql)}, `manifest "extra" has no paths.json row`},
		{"invalid manifest", map[string]string{"courses/sql/course.json": strings.Replace(sql, `"format": 1`, `"format": 3`, 1)}, "format: must be 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ms, err := course.Load(loadFS(t, tc.override))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Load: %v", err)
				}
				if len(ms) != 6 {
					t.Fatalf("got %d manifests, want 6", len(ms))
				}
				return
			}
			if err == nil {
				t.Fatalf("accepted; want an error mentioning %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error does not mention %q:\n%v", tc.want, err)
			}
		})
	}
}

func TestItemValidateRules(t *testing.T) {
	base := func(t *testing.T) map[string]any { return readJSONFile(t, dirFS("testdata/items"), "valid-code.json") }
	for _, tc := range []struct {
		name   string
		mutate func(m map[string]any)
		want   string
	}{
		{"prefixed id for other courses", func(m map[string]any) { m["course"] = "sql"; m["concepts"] = []any{} }, "must match <id_prefix>-NNN"},
		{"concept of another course", func(m map[string]any) { m["concepts"] = []any{"sql:concept:joins"} }, "must reference the item's own course"},
		{"licensed needs a license", func(m map[string]any) { at(m, "provenance")["origin"] = "licensed" }, "provenance.license: is required"},
		{"unregistered value type", func(m map[string]any) {
			at(m["parts"].([]any)[0].(map[string]any), "config", "signature")["returns"] = "Matrix"
		}, `"Matrix" is not a registered type`},
		{"code config field on a choice part", func(m map[string]any) {
			p := m["parts"].([]any)[0].(map[string]any)
			p["type"] = "choice"
		}, `is not allowed on a "choice" part`},
		{"grader input must be a part", func(m map[string]any) { m["grader"].([]any)[0].(map[string]any)["inputs"] = []any{"nope"} }, `"nope" is not a part id`},
		{"graded part must feed a step", func(m map[string]any) { m["grader"] = []any{} }, "feeds no grader step"},
		{"key step reads final parts", func(m map[string]any) { m["grader"].([]any)[0].(map[string]any)["kind"] = "key" }, "a key step reads final parts"},
		{"ai_rubric needs a rubric ref", func(m map[string]any) {
			m["parts"].([]any)[0].(map[string]any)["cadence"] = "final"
			m["grader"].([]any)[0].(map[string]any)["kind"] = "ai_rubric"
		}, "an ai_rubric step needs a rubric"},
		{"review stamp is a date", func(m map[string]any) { at(m, "review")["hints"] = "soon" }, "must be an ISO date"},
		{"sample ids unique", func(m map[string]any) {
			s := at(m["parts"].([]any)[0].(map[string]any), "config")["samples"].([]any)
			s[1].(map[string]any)["id"] = "s1"
		}, `duplicate sample "s1"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := base(t)
			tc.mutate(doc)
			it, err := course.DecodeItem(mustMarshal(t, doc))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			err = it.Validate()
			if err == nil {
				t.Fatalf("accepted; want an error mentioning %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error does not mention %q:\n%v", tc.want, err)
			}
		})
	}
}

func TestProbeValidateFor(t *testing.T) {
	ms := loadAll(t)
	doc := readJSONFile(t, dirFS("testdata/items"), "valid-probes.json")
	at(doc, "revision")["probes"].([]any)[0].(map[string]any)["band"] = "recall"
	at(doc, "revision")["probes"].([]any)[1].(map[string]any)["criterion"] = "explained_well"
	delete(doc, "solution_facts")
	it, err := course.DecodeItem(mustMarshal(t, doc))
	if err != nil {
		t.Fatal(err)
	}
	err = it.ValidateFor(ms["dsa"])
	for _, want := range []string{
		`"recall" is not a band format of course "dsa"`,
		`"explained_well" is not a criterion of band "resolve"`,
		`"public:solution_facts.complexity" does not resolve`,
	} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want an error mentioning %q, got %v", want, err)
		}
	}
}
