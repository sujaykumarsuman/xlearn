package canon_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/course/canon"
)

// itemPaths walks course.Item by reflection and returns every JSON leaf path in
// classify.go's syntax: struct fields by JSON name, "[]" for the elements of a slice of
// structs; a slice of scalars (or a json.RawMessage) is one leaf.
func itemPaths() []string {
	var out []string
	var walk func(t reflect.Type, prefix string)
	walk = func(t reflect.Type, prefix string) {
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
		switch {
		case t.Kind() == reflect.Struct:
			for i := 0; i < t.NumField(); i++ {
				f := t.Field(i)
				name, ok := jsonName(f)
				if !ok {
					continue
				}
				p := name
				if prefix != "" {
					p = prefix + "." + name
				}
				walk(f.Type, p)
			}
			return
		case t.Kind() == reflect.Slice:
			e := t.Elem()
			for e.Kind() == reflect.Pointer {
				e = e.Elem()
			}
			if e.Kind() == reflect.Struct {
				walk(e, prefix+"[]")
				return
			}
		}
		out = append(out, prefix)
	}
	walk(reflect.TypeOf(course.Item{}), "")
	sort.Strings(out)
	return out
}

func jsonName(f reflect.StructField) (string, bool) {
	if !f.IsExported() {
		return "", false
	}
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "-" {
		return "", false
	}
	if name == "" {
		name = f.Name
	}
	return name, true
}

// Every course.Item path is classified exactly once, and classify.go names no path the
// type does not have.
func TestClassificationCoversEveryItemField(t *testing.T) {
	paths := itemPaths()
	have := map[string]bool{}
	for _, p := range paths {
		have[p] = true
		contract, known := canon.IsContractPath(p)
		if !known {
			t.Errorf("course.Item path %q is not classified: classify the new field in canon/classify.go (contract or content-only)", p)
		}
		_ = contract
	}
	for _, p := range append(canon.ContractPaths(), canon.ContentOnlyPaths()...) {
		if !have[p] {
			t.Errorf("classify.go lists %q, which course.Item does not have", p)
		}
	}
	both := map[string]bool{}
	for _, p := range canon.ContractPaths() {
		both[p] = true
	}
	for _, p := range canon.ContentOnlyPaths() {
		if both[p] {
			t.Errorf("%q is classified both contract and content-only", p)
		}
	}
}

// The classification is honest: mutating any contract path moves contract_hash, mutating
// any content-only path does not, and every path moves content_hash.
func TestClassificationIsHonest(t *testing.T) {
	sections := fixtureSections()
	for _, p := range itemPaths() {
		base := richItem(t)
		edited := richItem(t)
		mutate(t, reflect.ValueOf(edited).Elem(), strings.Split(p, "."), p)

		bc, err := canon.ContractHash(base)
		if err != nil {
			t.Fatal(err)
		}
		ec, err := canon.ContractHash(edited)
		if err != nil {
			t.Fatal(err)
		}
		if bc == "" {
			t.Fatal("the rich item must have a contract")
		}
		contract, _ := canon.IsContractPath(p)
		if contract && bc == ec {
			t.Errorf("%s is classified contract but editing it does not move contract_hash", p)
		}
		if !contract && bc != ec {
			t.Errorf("%s is classified content-only but editing it moves contract_hash", p)
		}
		bh, err := canon.ContentHash(&course.ResolvedItem{Item: *base, Sections: sections})
		if err != nil {
			t.Fatal(err)
		}
		eh, err := canon.ContentHash(&course.ResolvedItem{Item: *edited, Sections: sections})
		if err != nil {
			t.Fatal(err)
		}
		if bh == eh {
			t.Errorf("editing %s does not move content_hash", p)
		}
	}
}

// richItem is a course.Item with every field populated (two elements in every slice of
// structs), deterministic, distinct values everywhere. Not a valid item: hashing does
// not validate.
func richItem(t *testing.T) *course.Item {
	t.Helper()
	it := &course.Item{}
	n := 0
	fill(t, reflect.ValueOf(it).Elem(), &n)
	return it
}

var rawType = reflect.TypeOf(json.RawMessage{})

func fill(t *testing.T, v reflect.Value, n *int) {
	t.Helper()
	*n++
	switch {
	case v.Type() == rawType:
		v.Set(reflect.ValueOf(json.RawMessage(fmt.Sprintf(`{"k":[%d,{"z":%d}]}`, *n, *n))))
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		v.Set(reflect.New(v.Type().Elem()))
		fill(t, v.Elem(), n)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				fill(t, v.Field(i), n)
			}
		}
	case reflect.Slice:
		s := reflect.MakeSlice(v.Type(), 2, 2)
		for i := 0; i < 2; i++ {
			fill(t, s.Index(i), n)
		}
		v.Set(s)
	case reflect.String:
		v.SetString(fmt.Sprintf("v%03d", *n))
	case reflect.Int:
		v.SetInt(int64(*n))
	case reflect.Float64:
		v.SetFloat(float64(*n) + 0.25)
	case reflect.Bool:
		v.SetBool(*n%2 == 0)
	default:
		t.Fatalf("fill: unsupported kind %s (extend the test)", v.Kind())
	}
}

// mutate changes the leaf at path (element 0 of every slice on the way).
func mutate(t *testing.T, v reflect.Value, segs []string, path string) {
	t.Helper()
	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if len(segs) == 0 {
		mutateLeaf(t, v, path)
		return
	}
	name, isSlice := strings.CutSuffix(segs[0], "[]")
	f := fieldByJSON(v, name)
	if !f.IsValid() {
		t.Fatalf("mutate %s: no field %q", path, name)
	}
	if isSlice {
		f = f.Index(0)
	}
	mutate(t, f, segs[1:], path)
}

func fieldByJSON(v reflect.Value, name string) reflect.Value {
	for i := 0; i < v.NumField(); i++ {
		if n, ok := jsonName(v.Type().Field(i)); ok && n == name {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

func mutateLeaf(t *testing.T, v reflect.Value, path string) {
	t.Helper()
	if v.Type() == rawType {
		v.Set(reflect.ValueOf(json.RawMessage(`{"k":"mutated"}`)))
		return
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(v.String() + "x")
	case reflect.Int:
		v.SetInt(v.Int() + 1000)
	case reflect.Float64:
		v.SetFloat(v.Float() + 0.5)
	case reflect.Bool:
		v.SetBool(!v.Bool())
	case reflect.Slice:
		mutateLeaf(t, v.Index(0), path)
	default:
		t.Fatalf("mutate %s: unsupported kind %s", path, v.Kind())
	}
}
