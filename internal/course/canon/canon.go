// Package canon is xLearn's ONE canonical-hash definition (canon@1; t1 §3.4, ADR-0027
// §2): sha256 over canonical bytes obtained by re-marshalling typed Go values. The
// curriculum seed, practice, judge and packlint all import it; nobody hashes content any
// other way. It is stdlib-only and imports internal/course and nothing else, so any
// service can use it without cycles.
//
// Sprint m1-09 created it with ContentHash; m3-01 added PolicyVersion, ContractHash,
// Prefix and the field classification (classify.go).
//
// The three hashes:
//   - PolicyVersion: the course manifest (copied onto attempts and mock sessions).
//   - ContentHash: the whole resolved public item, sidecars inlined (problem.content_hash).
//   - ContractHash: only what hidden data depends on (contract.go; problem.contract_hash,
//     the pack's accepts_contract_hashes). "" for a self-path item.
//
// Canonical rules (canon@1), pinned by testdata/vectors.golden.json:
//   - Typed structs are re-marshalled with encoding/json: struct fields in declaration
//     order, map keys sorted, SetEscapeHTML(false), no trailing newline. Author
//     whitespace and key order therefore never move a hash.
//   - Absent ≡ empty: an object member whose value is null, [] or {} (after this rule is
//     applied inside it) is dropped, so an author's choice between leaving a field out and
//     writing it empty never moves a hash. Array elements are never dropped.
//   - Numbers are normalized: an integer literal within int64 keeps its decimal digits
//     (-0 → 0); any other number is parsed as float64 and written with
//     strconv.FormatFloat(f, 'g', -1, 64), except that an integral value within int64 is
//     written as an integer (so 1e3 ≡ 1000 and 1e-6 ≡ 0.000001). NaN, ±Inf and integer
//     literals outside int64 are invalid.
//   - Open JSON values (an item sample's args and expected) are decoded with UseNumber
//     and re-encoded key-sorted with the number rule, then hashed verbatim: inside them
//     an empty array or object is data, never dropped. No map[string]any or
//     json.RawMessage is ever hashed as the author wrote it.
//   - Set-valued fields of the contract view are sorted (part, option, field, probe and
//     step ids, class ops, step inputs, asset refs); positional lists (param types) keep
//     their order. Sections are hashed in canonical order: stage (attempt → hint →
//     solution), then order, then language, then kind.
//   - Domain separation: sha256("xlearn.<kind>@1\n" + bytes), kind ∈ {policy, content,
//     contract}, encoded "sha256:<64 hex>", so hashes of different kinds never coincide
//     and a future canon@2 is explicit.
//
// Changing any byte this package produces for an existing input invalidates every
// stored hash and every pack's accepts_contract_hashes: that is canon@2, never an edit.
package canon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// Hash kinds (the domain-separation prefix).
const (
	KindPolicy   = "policy"
	KindContent  = "content"
	KindContract = "contract"
)

// PrefixLen is the number of hex characters humans see (Prefix).
const PrefixLen = 12

// Bytes returns the canonical JSON of a typed value (canon@1): encoding/json with HTML
// escaping off and no trailing newline. The hashes apply the absent ≡ empty and number
// rules on top (see the package doc).
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

// Prefix returns the first PrefixLen hex characters of a "sha256:<hex>" hash, for humans
// (the API's contract_hash_prefix, packlint messages). "" stays "".
func Prefix(h string) string {
	hx := strings.TrimPrefix(h, "sha256:")
	if len(hx) > PrefixLen {
		hx = hx[:PrefixLen]
	}
	return hx
}

// PolicyVersion is a course manifest's version: the whole typed manifest, canonical.
// Attempts and mock sessions copy it (m2-01); nobody hashes a manifest any other way.
func PolicyVersion(m *course.Manifest) (string, error) {
	if m == nil {
		return "", errors.New("canon: nil manifest")
	}
	b, err := canonical(m, nil)
	if err != nil {
		return "", err
	}
	return Sum(KindPolicy, b), nil
}

// ContentHash is an item's version: the whole resolved public item (item.json with its
// section sidecars and reference files inlined). The seed stores it as
// problem.content_hash; any edit to the item, a section body or a reference file
// changes it, formatting never does. The input is not modified.
func ContentHash(it *course.ResolvedItem) (string, error) {
	if it == nil {
		return "", errors.New("canon: nil item")
	}
	view, err := contentView(it)
	if err != nil {
		return "", err
	}
	b, err := canonical(view, isSampleValue)
	if err != nil {
		return "", err
	}
	return Sum(KindContent, b), nil
}

// isSampleValue marks the open JSON values of a resolved item: a sample's args and
// expected (already normalized by contentView). They are hashed verbatim.
func isSampleValue(path []string) bool {
	return len(path) == 7 && path[0] == "item" && path[1] == "parts" && path[3] == "config" &&
		path[4] == "samples" && (path[6] == "args" || path[6] == "expected")
}

// contentView copies the item with open JSON values normalized and sections sorted.
func contentView(in *course.ResolvedItem) (*course.ResolvedItem, error) {
	out := &course.ResolvedItem{Item: in.Item, References: in.References}
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

// canonical returns canon@1 bytes of a typed value: Bytes, then the absent ≡ empty and
// number rules (compact). Members at an opaque path are copied verbatim.
func canonical(v any, opaque func(path []string) bool) ([]byte, error) {
	b, err := Bytes(v)
	if err != nil {
		return nil, err
	}
	out, err := compact(b, opaque)
	if err != nil {
		return nil, fmt.Errorf("canon: %w", err)
	}
	return out, nil
}

// compact re-encodes JSON (as Bytes produced it) keeping member order: object members
// whose compacted value is null, [] or {} are dropped, numbers are normalized, strings
// are re-encoded as Bytes encodes them. Array elements are never dropped. A member whose
// path (object keys, "*" for an array element) satisfies opaque is copied verbatim and
// kept even when empty.
func compact(b []byte, opaque func(path []string) bool) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	c := &compactor{dec: dec, opaque: opaque}
	var out bytes.Buffer
	if err := c.value(nil, &out); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing data after the JSON value")
	}
	return out.Bytes(), nil
}

type compactor struct {
	dec    *json.Decoder
	opaque func(path []string) bool
}

func (c *compactor) value(path []string, out *bytes.Buffer) error {
	tok, err := c.dec.Token()
	if err != nil {
		return err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			return c.object(path, out)
		case '[':
			return c.array(path, out)
		}
		return fmt.Errorf("unexpected %q", t)
	case json.Number:
		n, err := normalizeNumber(string(t))
		if err != nil {
			return err
		}
		out.WriteString(string(n))
	case string:
		s, err := Bytes(t)
		if err != nil {
			return err
		}
		out.Write(s)
	case bool:
		out.WriteString(strconv.FormatBool(t))
	case nil:
		out.WriteString("null")
	default:
		return fmt.Errorf("unexpected token %T", tok)
	}
	return nil
}

func (c *compactor) object(path []string, out *bytes.Buffer) error {
	out.WriteByte('{')
	first := true
	for c.dec.More() {
		tok, err := c.dec.Token()
		if err != nil {
			return err
		}
		key, ok := tok.(string)
		if !ok {
			return fmt.Errorf("object key %v is not a string", tok)
		}
		p := append(path[:len(path):len(path)], key)
		var member bytes.Buffer
		keep := true
		if c.opaque != nil && c.opaque(p) {
			var raw json.RawMessage
			if err := c.dec.Decode(&raw); err != nil {
				return err
			}
			member.Write(raw)
		} else {
			if err := c.value(p, &member); err != nil {
				return err
			}
			keep = !isEmptyJSON(member.Bytes())
		}
		if !keep {
			continue
		}
		if !first {
			out.WriteByte(',')
		}
		first = false
		k, err := Bytes(key)
		if err != nil {
			return err
		}
		out.Write(k)
		out.WriteByte(':')
		out.Write(member.Bytes())
	}
	if _, err := c.dec.Token(); err != nil { // '}'
		return err
	}
	out.WriteByte('}')
	return nil
}

func (c *compactor) array(path []string, out *bytes.Buffer) error {
	out.WriteByte('[')
	p := append(path[:len(path):len(path)], "*")
	for i := 0; c.dec.More(); i++ {
		if i > 0 {
			out.WriteByte(',')
		}
		if err := c.value(p, out); err != nil {
			return err
		}
	}
	if _, err := c.dec.Token(); err != nil { // ']'
		return err
	}
	out.WriteByte(']')
	return nil
}

func isEmptyJSON(b []byte) bool {
	s := string(b)
	return s == "null" || s == "[]" || s == "{}"
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
