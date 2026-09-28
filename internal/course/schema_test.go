package course_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	// Test-only: a pure-Go JSON Schema 2020-12 validator, used here to prove the schema
	// files are valid and agree with the fixtures. Runtime validation stays the Go types
	// plus Validate; no production package imports this module.
	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// The item-schema freeze tests (sprint m1-01 task 3): 1 types⇄schema parity,
// 2 answer-free, 3 ADR-0029 field presence, 4 the freeze guard, 5 fixtures,
// 6 schema validity with a test-only JSON Schema validator.

const (
	itemSchemaPath   = "_schema/item.schema.json"
	courseSchemaPath = "_schema/course.schema.json"
	frozenItemSchema = "testdata/item.schema.v1.frozen.json"
)

var rawMessageType = reflect.TypeOf(json.RawMessage{})

// --- schema and Go-type trees -------------------------------------------------------

func readJSONFile(t *testing.T, fsys fs.FS, name string) map[string]any {
	t.Helper()
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return doc
}

func itemSchema(t *testing.T) map[string]any { return readJSONFile(t, curriculum.FS, itemSchemaPath) }
func courseSchema(t *testing.T) map[string]any {
	return readJSONFile(t, curriculum.FS, courseSchemaPath)
}

// contentSchemas are the m1-09 schemas of the per-course content files and the lock.
var contentSchemas = []string{"phases", "weeks", "concepts", "ids-lock", "paths"}

func contentSchema(t *testing.T, name string) map[string]any {
	return readJSONFile(t, curriculum.FS, "_schema/"+name+".schema.json")
}

// resolve follows local `#/$defs/<name>` refs.
func resolve(t *testing.T, doc, node map[string]any) map[string]any {
	t.Helper()
	for i := 0; i < 16; i++ {
		ref, ok := node["$ref"].(string)
		if !ok {
			return node
		}
		name, ok := strings.CutPrefix(ref, "#/$defs/")
		if !ok {
			t.Fatalf("unsupported $ref %q (only local #/$defs refs)", ref)
		}
		next, ok := doc["$defs"].(map[string]any)[name].(map[string]any)
		if !ok {
			t.Fatalf("dangling $ref %q", ref)
		}
		node = next
	}
	t.Fatal("$ref chain too deep")
	return nil
}

func joinPath(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// schemaPaths returns every property of the schema tree by full JSON path
// ("parts[].config.samples[].expected"), resolving local refs. Map values (patternProperties)
// appear under "<path>{}". Conditional blocks (if/then) only add requirements, so they are
// not walked.
func schemaPaths(t *testing.T, doc map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	var walk func(node map[string]any, prefix string, depth int)
	walk = func(node map[string]any, prefix string, depth int) {
		if depth > 32 {
			t.Fatalf("schema too deep at %q", prefix)
		}
		node = resolve(t, doc, node)
		if props, ok := node["properties"].(map[string]any); ok {
			for name, sub := range props {
				p := joinPath(prefix, name)
				out[p] = resolve(t, doc, sub.(map[string]any))
				walk(sub.(map[string]any), p, depth+1)
			}
		}
		if pp, ok := node["patternProperties"].(map[string]any); ok {
			for _, sub := range pp {
				walk(sub.(map[string]any), prefix+"{}", depth+1)
			}
		}
		if items, ok := node["items"].(map[string]any); ok {
			walk(items, prefix+"[]", depth+1)
		}
	}
	walk(doc, "", 0)
	return out
}

func jsonName(f reflect.StructField) string {
	if !f.IsExported() {
		return ""
	}
	tag := f.Tag.Get("json")
	if tag == "-" {
		return ""
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return f.Name
	}
	return name
}

func deref(rt reflect.Type) reflect.Type {
	for rt.Kind() == reflect.Pointer {
		rt = rt.Elem()
	}
	return rt
}

// goPaths returns every JSON field of the Go type tree by full JSON path, in the same
// notation as schemaPaths.
func goPaths(root reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	var walk func(rt reflect.Type, prefix string)
	walk = func(rt reflect.Type, prefix string) {
		rt = deref(rt)
		switch {
		case rt == rawMessageType:
		case rt.Kind() == reflect.Struct:
			for i := 0; i < rt.NumField(); i++ {
				f := rt.Field(i)
				name := jsonName(f)
				if name == "" {
					continue
				}
				p := joinPath(prefix, name)
				out[p] = f.Type
				walk(f.Type, p)
			}
		case rt.Kind() == reflect.Slice:
			walk(rt.Elem(), prefix+"[]")
		case rt.Kind() == reflect.Map:
			walk(rt.Elem(), prefix+"{}")
		}
	}
	walk(root, "")
	return out
}

var (
	itemType     = reflect.TypeOf(course.Item{})
	manifestType = reflect.TypeOf(course.Manifest{})
)

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// --- 1. types ⇄ schema parity ---------------------------------------------------------

func TestTypesSchemaParity(t *testing.T) {
	for _, tc := range []struct {
		name string
		doc  map[string]any
		rt   reflect.Type
	}{
		{"item", itemSchema(t), itemType},
		{"manifest", courseSchema(t), manifestType},
		// m1-09 content files.
		{"phases", contentSchema(t, "phases"), reflect.TypeOf([]course.Phase{})},
		{"weeks", contentSchema(t, "weeks"), reflect.TypeOf([]course.Week{})},
		{"concepts", contentSchema(t, "concepts"), reflect.TypeOf([]course.Concept{})},
		{"ids-lock", contentSchema(t, "ids-lock"), reflect.TypeOf(course.IDsLock{})},
		{"paths", contentSchema(t, "paths"), reflect.TypeOf([]course.PathRow{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sp := schemaPaths(t, tc.doc)
			gp := goPaths(tc.rt)
			for _, p := range sortedKeys(gp) {
				if _, ok := sp[p]; !ok {
					t.Errorf("Go field %q has no schema property", p)
				}
			}
			for _, p := range sortedKeys(sp) {
				if _, ok := gp[p]; !ok {
					t.Errorf("schema property %q has no Go field", p)
				}
			}
			for _, p := range sortedKeys(gp) {
				if node, ok := sp[p]; ok {
					if msg := typeMismatch(node, gp[p]); msg != "" {
						t.Errorf("%s: %s", p, msg)
					}
				}
			}
		})
	}
}

// typeMismatch reports a JSON-type disagreement between a schema node and a Go type.
func typeMismatch(node map[string]any, rt reflect.Type) string {
	rt = deref(rt)
	if rt == rawMessageType {
		return ""
	}
	var want string
	switch rt.Kind() {
	case reflect.Struct:
		want = "object"
	case reflect.Map:
		want = "object"
		if _, ok := node["patternProperties"]; !ok {
			return "a Go map needs patternProperties in the schema"
		}
	case reflect.Slice:
		want = "array"
	case reflect.String:
		want = "string"
	case reflect.Int:
		want = "integer"
	case reflect.Float64:
		want = "number"
	case reflect.Bool:
		want = "boolean"
	default:
		return fmt.Sprintf("unexpected Go kind %s", rt.Kind())
	}
	if typ, ok := node["type"].(string); ok {
		if typ == want || (want == "number" && typ == "integer") {
			return ""
		}
		return fmt.Sprintf("schema type %q, Go wants %q", typ, want)
	}
	// enum / const nodes carry values instead of a type.
	var vals []any
	if e, ok := node["enum"].([]any); ok {
		vals = e
	}
	if c, ok := node["const"]; ok {
		vals = []any{c}
	}
	if len(vals) == 0 {
		return "schema node has no type, enum or const"
	}
	for _, v := range vals {
		if got := jsonKind(v); got != want && (want != "integer" || got != "number") {
			return fmt.Sprintf("enum/const value %v is %s, Go wants %s", v, got, want)
		}
	}
	return ""
}

func jsonKind(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return "null"
}

// Every object shape in both schemas is closed: additionalProperties false. Conditional
// blocks (if/then/else) only add requirements, so they are exempt.
func TestSchemasCloseEveryObject(t *testing.T) {
	docs := map[string]map[string]any{"item": itemSchema(t), "course": courseSchema(t)}
	for _, name := range contentSchemas {
		docs[name] = contentSchema(t, name)
	}
	for name, doc := range docs {
		var visit func(node map[string]any, at string)
		visit = func(node map[string]any, at string) {
			_, hasProps := node["properties"]
			_, hasPattern := node["patternProperties"]
			if node["type"] == "object" || hasProps || hasPattern {
				if ap, ok := node["additionalProperties"].(bool); !ok || ap {
					t.Errorf("%s schema %s: object without additionalProperties: false", name, at)
				}
			}
			for _, kw := range []string{"properties", "patternProperties", "$defs"} {
				if m, ok := node[kw].(map[string]any); ok {
					for k, sub := range m {
						visit(sub.(map[string]any), at+"/"+kw+"/"+k)
					}
				}
			}
			if items, ok := node["items"].(map[string]any); ok {
				visit(items, at+"/items")
			}
			for _, kw := range []string{"allOf", "anyOf", "oneOf"} {
				if arr, ok := node[kw].([]any); ok {
					for i, sub := range arr {
						visit(sub.(map[string]any), fmt.Sprintf("%s/%s/%d", at, kw, i))
					}
				}
			}
		}
		visit(doc, "#")
	}
}

// --- 2. answer-free -------------------------------------------------------------------

// Property names that could hold an answer (ADR-0027 §1). A prefix entry matches any
// name that starts with it (e.g. "key" matches key_source and keys).
var (
	answerPrefixes = []string{"answer", "correct", "expected", "hidden", "secret", "anchor", "exemplar", "red_flag", "solution", "key"}
	answerExact    = []string{"must_cover", "cases", "tests", "rationale"}
)

// answerAllowlist names, by FULL JSON path, the only places a denylisted name may appear,
// each with its reason. The same name anywhere else still fails.
var answerAllowlist = map[string]map[string]string{
	"item": {
		"parts[].config.samples[].expected": "public sample output, shown with the statement",
		"solution_facts":                    "public complexity facts, served only with the solution stage",
		"revision.probes[].key_source":      "names where a key lives (public:<field> or pack), never a key; pattern-checked",
	},
	"manifest": {
		"stages.solution":                 "the solution stage's CONFIG ({label} only), not solution content",
		"revision.bands[].criteria[].key": "a criterion identifier (a check name), never an expected value; pattern-checked",
	},
}

func deniedName(name string) bool {
	for _, p := range answerPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	for _, e := range answerExact {
		if name == e {
			return true
		}
	}
	return false
}

func lastSegment(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		path = path[i+1:]
	}
	return strings.TrimSuffix(strings.TrimSuffix(path, "{}"), "[]")
}

func TestAnswerFree(t *testing.T) {
	trees := []struct {
		kind, name string
		paths      []string
	}{
		{"item", "item schema", sortedKeys(schemaPaths(t, itemSchema(t)))},
		{"item", "item Go types", sortedKeys(goPaths(itemType))},
		{"manifest", "manifest schema", sortedKeys(schemaPaths(t, courseSchema(t)))},
		{"manifest", "manifest Go types", sortedKeys(goPaths(manifestType))},
	}
	for _, tree := range trees {
		seen := map[string]bool{}
		for _, p := range tree.paths {
			seen[p] = true
			if !deniedName(lastSegment(p)) {
				continue
			}
			if reason, ok := answerAllowlist[tree.kind][p]; ok {
				t.Logf("%s: %s allowed (%s)", tree.name, p, reason)
				continue
			}
			t.Errorf("%s: property %q could hold an answer (ADR-0027 §1); answers live in the private eval pack", tree.name, p)
		}
		// A stale allowlist entry would hide nothing today but could mask a future rename.
		for p := range answerAllowlist[tree.kind] {
			if !seen[p] {
				t.Errorf("%s: allowlisted path %q no longer exists; drop it", tree.name, p)
			}
		}
	}

	// stages.solution is the stage's config: {label} and nothing else, in schema and Go.
	sol := schemaPaths(t, courseSchema(t))["stages.solution"]
	if got := sortedKeys(sol["properties"].(map[string]any)); !reflect.DeepEqual(got, []string{"label"}) {
		t.Errorf("manifest schema stages.solution has properties %v, want only [label]", got)
	}
	var goSol []string
	for p := range goPaths(manifestType) {
		if rest, ok := strings.CutPrefix(p, "stages.solution."); ok {
			goSol = append(goSol, rest)
		}
	}
	if !reflect.DeepEqual(goSol, []string{"label"}) {
		t.Errorf("course.SolutionStage has fields %v, want only [label]", goSol)
	}

	// The two allowlisted names that carry a value are pattern-locked to non-answers.
	for _, pc := range []struct {
		doc     map[string]any
		path    string
		pattern string
	}{
		{courseSchema(t), "revision.bands[].criteria[].key", `^[a-z][a-z0-9_]*$`},
		{itemSchema(t), "revision.probes[].key_source", `^(public:[a-z_.]+|pack)$`},
	} {
		if got := schemaPaths(t, pc.doc)[pc.path]["pattern"]; got != pc.pattern {
			t.Errorf("%s pattern = %v, want %q", pc.path, got, pc.pattern)
		}
	}
}

// --- 3. ADR-0029 / t4 §11.4 #19 content fields ----------------------------------------

func TestADR0029FieldsPresent(t *testing.T) {
	for _, tc := range []struct {
		kind  string
		doc   map[string]any
		rt    reflect.Type
		paths []string
	}{
		{"item", itemSchema(t), itemType, []string{"concepts"}},
		{"manifest", courseSchema(t), manifestType, []string{
			"mistakes.prefill",
			"revision.bands[].parts",
			"revision.bands[].criteria",
			"revision.bands[].mock_mode",
			"revision.drills",
			"mock.evidence_hints",
		}},
	} {
		sp, gp := schemaPaths(t, tc.doc), goPaths(tc.rt)
		for _, p := range tc.paths {
			if _, ok := sp[p]; !ok {
				t.Errorf("%s schema lacks ADR-0029 field %q", tc.kind, p)
			}
			if _, ok := gp[p]; !ok {
				t.Errorf("%s Go types lack ADR-0029 field %q", tc.kind, p)
			}
		}
	}
}

// --- 4. freeze guard ------------------------------------------------------------------

// TestItemSchemaFreeze: the live item schema must be an ADDITIVE superset of the snapshot
// taken at ev-schema-freeze — no removed property or $def, no type narrowing, no new
// required field, no removed enum value, no tightened bound, no changed pattern or const.
// New optional properties are fine. A genuinely breaking change needs an ADR amendment
// (ADR-0027) and a deliberate, reviewed update of the snapshot.
func TestItemSchemaFreeze(t *testing.T) {
	frozen := readJSONFile(t, os.DirFS("."), frozenItemSchema)
	if v := freezeViolations(frozen, itemSchema(t), "#"); len(v) > 0 {
		t.Fatalf("item schema v1 is frozen; these changes are not additive:\n  %s", strings.Join(v, "\n  "))
	}
}

// TestFreezeGuardCatchesBreakingChanges proves the guard itself: each mutation of the
// frozen schema must be reported.
func TestFreezeGuardCatchesBreakingChanges(t *testing.T) {
	frozen := readJSONFile(t, os.DirFS("."), frozenItemSchema)
	props := func(doc map[string]any) map[string]any { return doc["properties"].(map[string]any) }
	defs := func(doc map[string]any, name string) map[string]any {
		return doc["$defs"].(map[string]any)[name].(map[string]any)
	}
	for _, tc := range []struct {
		name   string
		mutate func(doc map[string]any)
		ok     bool
	}{
		{"removed property", func(d map[string]any) { delete(props(d), "concepts") }, false},
		{"new required field", func(d map[string]any) { d["required"] = append(d["required"].([]any), "concepts") }, false},
		{"removed enum value", func(d map[string]any) { props(d)["difficulty"].(map[string]any)["enum"] = []any{"easy", "hard"} }, false},
		{"type narrowed", func(d map[string]any) { props(d)["week_n"].(map[string]any)["type"] = "string" }, false},
		{"bound tightened", func(d map[string]any) { props(d)["concepts"].(map[string]any)["maxItems"] = 2.0 }, false},
		{"pattern changed", func(d map[string]any) { props(d)["course"].(map[string]any)["pattern"] = "^[a-z]+$" }, false},
		{"nested property removed", func(d map[string]any) { delete(defs(d, "provenance")["properties"].(map[string]any), "license") }, false},
		{"$def removed", func(d map[string]any) { delete(d["$defs"].(map[string]any), "probe") }, false},
		{"new optional property", func(d map[string]any) { props(d)["tags"] = map[string]any{"type": "array"} }, true},
		{"new enum value", func(d map[string]any) {
			defs(d, "part")["properties"].(map[string]any)["type"].(map[string]any)["enum"] = []any{"code", "text", "choice", "blank", "canvas", "audio"}
		}, true},
		{"bound loosened", func(d map[string]any) { props(d)["title"].(map[string]any)["maxLength"] = 300.0 }, true},
		{"comment edited", func(d map[string]any) { d["$comment"] = "reworded" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			live := deepCopy(t, frozen)
			tc.mutate(live)
			v := freezeViolations(frozen, live, "#")
			if tc.ok && len(v) > 0 {
				t.Errorf("an additive change was reported: %v", v)
			}
			if !tc.ok && len(v) == 0 {
				t.Error("a breaking change was not reported")
			}
		})
	}
}

func deepCopy(t *testing.T, doc map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// freezeViolations compares one frozen schema node with its live counterpart.
func freezeViolations(frozen, live map[string]any, at string) []string {
	var out []string
	add := func(format string, args ...any) { out = append(out, at+": "+fmt.Sprintf(format, args...)) }
	annotations := map[string]bool{"$schema": true, "$id": true, "$comment": true, "title": true, "description": true}

	for kw, fv := range frozen {
		if annotations[kw] {
			continue
		}
		lv, present := live[kw]
		switch kw {
		case "properties", "$defs", "patternProperties":
			fm := fv.(map[string]any)
			lm, _ := lv.(map[string]any)
			for name, fsub := range fm {
				lsub, ok := lm[name].(map[string]any)
				if !ok {
					add("%s %q was removed", kw, name)
					continue
				}
				out = append(out, freezeViolations(fsub.(map[string]any), lsub, at+"/"+kw+"/"+name)...)
			}
		case "items":
			lsub, ok := lv.(map[string]any)
			if !ok {
				add("items changed shape")
				continue
			}
			out = append(out, freezeViolations(fv.(map[string]any), lsub, at+"/items")...)
		case "required":
			// Handled below (live ⊆ frozen).
		case "type":
			if !present {
				continue // dropping a type constraint widens
			}
			if !superset(asList(lv), asList(fv)) {
				add("type narrowed from %v to %v", fv, lv)
			}
		case "enum":
			if !present {
				continue
			}
			if !superset(lv.([]any), fv.([]any)) {
				add("enum lost values: frozen %v, live %v", fv, lv)
			}
		case "minimum", "exclusiveMinimum", "minLength", "minItems":
			if present && lv.(float64) > fv.(float64) {
				add("%s tightened from %v to %v", kw, fv, lv)
			}
		case "maximum", "exclusiveMaximum", "maxLength", "maxItems":
			if present && lv.(float64) < fv.(float64) {
				add("%s tightened from %v to %v", kw, fv, lv)
			}
		case "additionalProperties":
			if fb, ok := fv.(bool); ok && fb {
				if lb, ok := lv.(bool); ok && !lb {
					add("additionalProperties closed")
				}
			}
		case "uniqueItems":
			if fv == false && lv == true {
				add("uniqueItems added")
			}
		default: // $ref, const, pattern, format and any other keyword must be unchanged.
			if !reflect.DeepEqual(fv, lv) {
				add("%s changed from %v to %v", kw, fv, lv)
			}
		}
	}
	// No new required field, and no new narrowing keyword the frozen node lacked.
	frozenReq := map[string]bool{}
	for _, r := range asList(frozen["required"]) {
		frozenReq[r.(string)] = true
	}
	for _, r := range asList(live["required"]) {
		if !frozenReq[r.(string)] {
			add("new required field %q", r)
		}
	}
	narrowing := []string{
		"type", "enum", "const", "pattern", "format", "minimum", "exclusiveMinimum", "maximum", "exclusiveMaximum",
		"minLength", "maxLength", "minItems", "maxItems", "uniqueItems", "allOf", "anyOf", "oneOf", "not", "if", "$ref",
	}
	for _, kw := range narrowing {
		if _, inLive := live[kw]; inLive {
			if _, inFrozen := frozen[kw]; !inFrozen {
				if kw == "uniqueItems" && live[kw] == false {
					continue
				}
				add("new constraint %q", kw)
			}
		}
	}
	return out
}

func asList(v any) []any {
	switch x := v.(type) {
	case nil:
		return nil
	case []any:
		return x
	default:
		return []any{x}
	}
}

func superset(big, small []any) bool {
	for _, s := range small {
		found := false
		for _, b := range big {
			if reflect.DeepEqual(b, s) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// --- 5. fixtures ----------------------------------------------------------------------

// validItemFixtures must decode, validate, and validate against their course manifest.
var validItemFixtures = []string{"valid-self.json", "valid-code.json", "valid-probes.json", "valid-structured.json"}

// invalidItemFixtures must fail, for the stated reason.
var invalidItemFixtures = map[string]string{
	"invalid-unknown-field.json":              `unknown field "statement_md"`,
	"invalid-origin-copied.json":              `"copied" is rejected`,
	"invalid-http-link.json":                  "must be an https:// URL",
	"invalid-four-concepts.json":              "at most 3 concepts",
	"invalid-malformed-id.json":               "DSA ids are bare numbers",
	"invalid-answer-on-option.json":           `unknown field "answer"`,
	"invalid-correct-flag.json":               `unknown field "correct"`,
	"invalid-hidden-tests.json":               `unknown field "hidden_tests"`,
	"invalid-key-source-value.json":           "key_source",
	"../manifests/invalid-solution-code.json": `unknown field "code"`,
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "items", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestItemFixtures(t *testing.T) {
	ms := loadAll(t)
	for _, name := range validItemFixtures {
		t.Run(name, func(t *testing.T) {
			it, err := course.DecodeItem(readFixture(t, name))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			m := ms[it.Course]
			if m == nil {
				t.Fatalf("no manifest for course %q", it.Course)
			}
			if err := it.ValidateFor(m); err != nil {
				t.Fatalf("ValidateFor(%s): %v", it.Course, err)
			}
		})
	}
	for name, reason := range invalidItemFixtures {
		t.Run(name, func(t *testing.T) {
			b := readFixture(t, name)
			var err error
			if strings.Contains(name, "manifests/") {
				var m *course.Manifest
				if m, err = course.DecodeManifest(b); err == nil {
					err = m.Validate()
				}
			} else {
				var it *course.Item
				if it, err = course.DecodeItem(b); err == nil {
					err = it.Validate()
				}
			}
			if err == nil {
				t.Fatalf("accepted; want an error mentioning %q", reason)
			}
			if !strings.Contains(err.Error(), reason) {
				t.Fatalf("error %q does not mention %q", err, reason)
			}
		})
	}
}

// --- 6. schema validity (test-only JSON Schema validator) -----------------------------

func compileSchema(t *testing.T, fsys fs.FS, name string) *jsonschema.Schema {
	t.Helper()
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	id, _ := doc.(map[string]any)["$id"].(string)
	if id == "" {
		t.Fatalf("%s has no $id", name)
	}
	c := jsonschema.NewCompiler()
	c.AssertFormat()
	if err := c.AddResource(id, doc); err != nil {
		t.Fatalf("add %s: %v", name, err)
	}
	// Compile also validates the schema against the 2020-12 meta-schema.
	sch, err := c.Compile(id)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	return sch
}

func validateAgainst(t *testing.T, sch *jsonschema.Schema, b []byte) error {
	t.Helper()
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("decode instance: %v", err)
	}
	return sch.Validate(inst)
}

func TestSchemaValidity(t *testing.T) {
	itemSch := compileSchema(t, curriculum.FS, itemSchemaPath)
	courseSch := compileSchema(t, curriculum.FS, courseSchemaPath)
	compileSchema(t, os.DirFS("."), frozenItemSchema)

	for _, name := range validItemFixtures {
		if err := validateAgainst(t, itemSch, readFixture(t, name)); err != nil {
			t.Errorf("%s: rejected by item.schema.json: %v", name, err)
		}
	}
	for name := range invalidItemFixtures {
		sch := itemSch
		if strings.Contains(name, "manifests/") {
			sch = courseSch
		}
		if err := validateAgainst(t, sch, readFixture(t, name)); err == nil {
			t.Errorf("%s: accepted by the JSON Schema; want a rejection", name)
		}
	}

	names, err := fs.Glob(curriculum.FS, course.ManifestGlob)
	if err != nil || len(names) == 0 {
		t.Fatalf("glob manifests: %v (%d)", err, len(names))
	}
	for _, name := range names {
		b, err := fs.ReadFile(curriculum.FS, name)
		if err != nil {
			t.Fatal(err)
		}
		if err := validateAgainst(t, courseSch, b); err != nil {
			t.Errorf("%s: rejected by course.schema.json: %v", name, err)
		}
	}
}
