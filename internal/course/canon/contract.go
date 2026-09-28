package canon

import (
	"errors"
	"sort"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// The contract view: exactly what hidden data depends on (ADR-0027 §2, t1 §3.4,
// t4 §5.6 #9). classify.go lists every course.Item JSON path on one side or the other;
// its test fails when a new schema field is left unclassified.
//
// In contract_hash: every part's {id, type}; a code part's signature (mode, name,
// ordered param types, returns, class ops sorted by name), harness name@v and checker
// {name, params}; a choice's mode and option ids; blank/text field ids and blank field
// kinds; a canvas part's palette id@v; grader steps {step, kind, sorted inputs, rubric
// id@v}; every probe's {id, type, key_source} with its choice/blank mode, option ids,
// field ids and kinds; asset id@v refs.
//
// Content-only (content_hash alone): prompts, statements, sections, labels (option and
// field labels, param names), samples, limits, languages, constraints (packlint's
// advisory re-check covers them), cadence, required flags, weights, provenance, links,
// review stamps, concepts, solution facts, probe band / criterion / timer / prompt, and
// all other metadata. A part's grading enters only through the self-path gate.
//
// Two deliberate gaps against t4 §5.6 #9: generator @v is not here (generator specs
// live in the pack's item pack.json, not the public item), and constraints[] is
// content-only (decided in m3-01; the owner may revisit it).

type contractView struct {
	Parts  []contractPart  `json:"parts,omitempty"`
	Grader []contractStep  `json:"grader,omitempty"`
	Probes []contractProbe `json:"probes,omitempty"`
	Assets []string        `json:"assets,omitempty"`
}

type contractPart struct {
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Signature *contractSignature `json:"signature,omitempty"`
	Harness   string             `json:"harness,omitempty"`
	Checker   *contractChecker   `json:"checker,omitempty"`
	Mode      string             `json:"mode,omitempty"`
	Options   []string           `json:"options,omitempty"`
	Fields    []contractField    `json:"fields,omitempty"`
	Palette   string             `json:"palette,omitempty"`
}

type contractSignature struct {
	Mode string `json:"mode,omitempty"`
	Name string `json:"name,omitempty"`
	// Params are the parameter TYPES in order (names are labels).
	Params  []string     `json:"params,omitempty"`
	Returns string       `json:"returns,omitempty"`
	Ops     []contractOp `json:"ops,omitempty"`
}

type contractOp struct {
	Name    string   `json:"name,omitempty"`
	Params  []string `json:"params,omitempty"`
	Returns string   `json:"returns,omitempty"`
}

type contractChecker struct {
	Name   string                 `json:"name,omitempty"`
	Params *contractCheckerParams `json:"params,omitempty"`
}

type contractCheckerParams struct {
	Eps float64 `json:"eps,omitempty"`
}

type contractField struct {
	ID   string `json:"id"`
	Kind string `json:"kind,omitempty"`
}

type contractStep struct {
	Step   string   `json:"step"`
	Kind   string   `json:"kind,omitempty"`
	Inputs []string `json:"inputs,omitempty"`
	Rubric string   `json:"rubric,omitempty"`
}

type contractProbe struct {
	ID        string          `json:"id"`
	Type      string          `json:"type,omitempty"`
	KeySource string          `json:"key_source,omitempty"`
	Mode      string          `json:"mode,omitempty"`
	Options   []string        `json:"options,omitempty"`
	Fields    []contractField `json:"fields,omitempty"`
}

// IsSelfPath reports whether nothing in the item depends on hidden data: no part is
// graded (grading "none" everywhere, or no parts) and no probe's key lives in the pack.
// Such an item has no contract: ContractHash is "".
func IsSelfPath(it *course.Item) bool {
	for _, p := range it.Parts {
		if p.Grading != "none" {
			return false
		}
	}
	if it.Revision != nil {
		for _, pr := range it.Revision.Probes {
			if pr.KeySource == "pack" {
				return false
			}
		}
	}
	return true
}

// ContractHash is the item's grading contract: the hash a pack lists in
// accepts_contract_hashes and an attempt pins. It is "" for a self-path item
// (IsSelfPath). The input is not modified.
func ContractHash(it *course.Item) (string, error) {
	if it == nil {
		return "", errors.New("canon: nil item")
	}
	if IsSelfPath(it) {
		return "", nil
	}
	b, err := ContractBytes(it)
	if err != nil {
		return "", err
	}
	return Sum(KindContract, b), nil
}

// ContractBytes returns the canonical bytes ContractHash hashes (for tests, golden
// vectors and packlint's explanations). It ignores the self-path gate.
func ContractBytes(it *course.Item) ([]byte, error) {
	if it == nil {
		return nil, errors.New("canon: nil item")
	}
	return canonical(contractOf(it), nil)
}

func contractOf(it *course.Item) contractView {
	var v contractView
	for _, p := range it.Parts {
		v.Parts = append(v.Parts, contractPartOf(p))
	}
	sort.SliceStable(v.Parts, func(i, j int) bool { return v.Parts[i].ID < v.Parts[j].ID })
	for _, s := range it.Grader {
		v.Grader = append(v.Grader, contractStep{
			Step: s.Step, Kind: s.Kind, Inputs: sortedCopy(s.Inputs), Rubric: s.Rubric,
		})
	}
	sort.SliceStable(v.Grader, func(i, j int) bool { return v.Grader[i].Step < v.Grader[j].Step })
	if it.Revision != nil {
		for _, pr := range it.Revision.Probes {
			cp := contractProbe{ID: pr.ID, Type: pr.Type, KeySource: pr.KeySource}
			if c := pr.Config; c != nil {
				cp.Mode = c.Mode
				cp.Options = optionIDs(c.Options)
				cp.Fields = fieldsOf(c.Fields)
			}
			v.Probes = append(v.Probes, cp)
		}
		sort.SliceStable(v.Probes, func(i, j int) bool { return v.Probes[i].ID < v.Probes[j].ID })
	}
	v.Assets = sortedCopy(it.Assets)
	return v
}

func contractPartOf(p course.Part) contractPart {
	c := p.Config
	cp := contractPart{
		ID: p.ID, Type: p.Type, Harness: c.Harness, Mode: c.Mode, Palette: c.Palette,
		Options: optionIDs(c.Options), Fields: fieldsOf(c.Fields),
	}
	if s := c.Signature; s != nil {
		sig := &contractSignature{Mode: s.Mode, Name: s.Name, Params: paramTypes(s.Params), Returns: s.Returns}
		for _, op := range s.Ops {
			sig.Ops = append(sig.Ops, contractOp{Name: op.Name, Params: paramTypes(op.Params), Returns: op.Returns})
		}
		sort.SliceStable(sig.Ops, func(i, j int) bool { return sig.Ops[i].Name < sig.Ops[j].Name })
		cp.Signature = sig
	}
	if ch := c.Checker; ch != nil {
		cc := &contractChecker{Name: ch.Name}
		if ch.Params != nil {
			cc.Params = &contractCheckerParams{Eps: ch.Params.Eps}
		}
		cp.Checker = cc
	}
	return cp
}

func paramTypes(ps []course.Param) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Type)
	}
	return out
}

func optionIDs(os []course.Option) []string {
	var out []string
	for _, o := range os {
		out = append(out, o.ID)
	}
	sort.Strings(out)
	return out
}

func fieldsOf(fs []course.Field) []contractField {
	var out []contractField
	for _, f := range fs {
		out = append(out, contractField{ID: f.ID, Kind: f.Kind})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedCopy(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
