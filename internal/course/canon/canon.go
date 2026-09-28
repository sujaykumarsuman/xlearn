// Package canon is xLearn's ONE canonical-hash definition (canon@1; t1 §3.4, ADR-0027
// §2): sha256 over canonical bytes obtained by re-marshalling typed Go values. The
// curriculum seed, practice, judge and packlint all import it; nobody hashes content any
// other way. It is stdlib-only and imports internal/course and nothing else, so any
// service can use it without cycles.
//
// Sprint m1-09 created it with ContentHash; m3-01 extends it (Bytes' rules stay, and it
// adds PolicyVersion, ContractHash and Prefix).
//
// Canonical rules (canon@1):
//   - Typed structs are re-marshalled with encoding/json: struct fields in declaration
//     order, map keys sorted, SetEscapeHTML(false), no trailing newline. Author
//     whitespace and key order therefore never move a hash.
//   - Open JSON values (an item sample's args and expected) are decoded with UseNumber
//     and re-encoded key-sorted with numbers normalized: an integer literal within int64
//     keeps its decimal digits (-0 → 0); any other number is parsed as float64 and
//     written with strconv.FormatFloat(f, 'g', -1, 64), except that an integral value
//     within int64 is written as an integer (so 1e3 ≡ 1000 and 1e-6 ≡ 0.000001). An
//     integer literal outside int64 is invalid.
//   - Sections are hashed in canonical order: stage (attempt → hint → solution), then
//     order, then language, then kind.
//   - Domain separation: sha256("xlearn.<kind>@1\n" + bytes), encoded "sha256:<64 hex>",
//     so hashes of different kinds never coincide.
package canon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Hash kinds (the domain-separation prefix).
const (
	KindContent = "content"
)

// Bytes returns the canonical JSON of a typed value (canon@1): encoding/json with HTML
// escaping off and no trailing newline.
func Bytes(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, fmt.Errorf("canon: marshal: %w", err)
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// Sum hashes canonical bytes under a domain kind: "sha256:<64 hex>".
func Sum(kind string, b []byte) string {
	h := sha256.New()
	h.Write([]byte("xlearn." + kind + "@1\n"))
	h.Write(b)
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

// ContentHash is an item's version: the whole resolved public item (item.json with its
// section sidecars inlined). The seed stores it as problem.content_hash; any edit to the
// item or a section body changes it, formatting never does. The input is not modified.
func ContentHash(it *course.ResolvedItem) (string, error) {
	if it == nil {
		return "", errors.New("canon: nil item")
	}
	view, err := contentView(it)
	if err != nil {
		return "", err
	}
	b, err := Bytes(view)
	if err != nil {
		return "", err
	}
	return Sum(KindContent, b), nil
}

// contentView copies the item with open JSON values normalized and sections sorted.
func contentView(in *course.ResolvedItem) (*course.ResolvedItem, error) {
	out := &course.ResolvedItem{Item: in.Item}
	if len(in.Item.Parts) > 0 {
		out.Item.Parts = make([]course.Part, len(in.Item.Parts))
		for i, p := range in.Item.Parts {
			if len(p.Config.Samples) > 0 {
				samples := make([]course.Sample, len(p.Config.Samples))
				for j, s := range p.Config.Samples {
					args := make([]json.RawMessage, len(s.Args))
					for k, a := range s.Args {
						n, err := NormalizeJSON(a)
						if err != nil {
							return nil, fmt.Errorf("canon: parts[%d].config.samples[%d].args[%d]: %w", i, j, k, err)
						}
						args[k] = n
					}
					exp, err := NormalizeJSON(s.Expected)
					if err != nil {
						return nil, fmt.Errorf("canon: parts[%d].config.samples[%d].expected: %w", i, j, err)
					}
					s.Args, s.Expected = args, exp
					samples[j] = s
				}
				p.Config.Samples = samples
			}
			out.Item.Parts[i] = p
		}
	}
	out.Sections = append([]course.Section(nil), in.Sections...)
	sort.SliceStable(out.Sections, func(i, j int) bool {
		a, b := out.Sections[i], out.Sections[j]
		if ra, rb := course.StageRank(a.Stage), course.StageRank(b.Stage); ra != rb {
			return ra < rb
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		if a.Language != b.Language {
			return a.Language < b.Language
		}
		return a.Kind < b.Kind
	})
	return out, nil
}

// NormalizeJSON re-encodes an open JSON value canonically: keys sorted, insignificant
// whitespace dropped, numbers normalized (see the package doc). An empty input stays
// empty (an absent value).
func NormalizeJSON(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	if dec.More() {
		return nil, errors.New("trailing data after the JSON value")
	}
	nv, err := normalizeValue(v)
	if err != nil {
		return nil, err
	}
	return Bytes(nv)
}

func normalizeValue(v any) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		for k, e := range t {
			n, err := normalizeValue(e)
			if err != nil {
				return nil, err
			}
			t[k] = n
		}
		return t, nil
	case []any:
		for i, e := range t {
			n, err := normalizeValue(e)
			if err != nil {
				return nil, err
			}
			t[i] = n
		}
		return t, nil
	case json.Number:
		return normalizeNumber(string(t))
	default:
		return v, nil
	}
}

var intLiteralRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)

// normalizeNumber applies canon@1's number rule.
func normalizeNumber(s string) (json.Number, error) {
	if intLiteralRe.MatchString(s) {
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return "", fmt.Errorf("integer %s is outside int64", s)
		}
		return json.Number(strconv.FormatInt(n, 10)), nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return "", fmt.Errorf("number %s is not a finite float64", s)
	}
	if f == math.Trunc(f) && f >= -(1<<63) && f < (1<<63) {
		return json.Number(strconv.FormatInt(int64(f), 10)), nil
	}
	return json.Number(strconv.FormatFloat(f, 'g', -1, 64)), nil
}
