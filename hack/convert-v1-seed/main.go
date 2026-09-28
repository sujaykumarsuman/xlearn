// Command convert-v1-seed is m1-09's ONE-SHOT converter: it reads the v1 DSA seed
// (curriculum/dsa/{phases,weeks,concepts,problems}.json) and writes the per-course
// layout under curriculum/courses/dsa/ (ADR-0027 §2, t1 §3.2) plus curriculum/ids.lock.json.
// It runs once; its output is committed with it, and it is deleted later in the same PR
// (its final source stays reachable at the commit recorded in docs/v2/status.md).
//
//	go run ./hack/convert-v1-seed            # from the repo root
//
// Byte-exactness is the contract: every body and code fragment is written exactly as v1
// stores it (no trailing newline added, no reformatting), and the row snapshot
// (internal/curriculum/testdata/v1-seed-snapshot.json, committed before this converter)
// proves the new loader reproduces every v1 row. v1's code is display fragments (no
// package clause, 4-space indents), so it is written as *.go.snip under _code/, never as
// .go: the repo-wide `gofmt -l .` walks `_` directories and would fail on a fragment.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/course"
)

// v1 seed rows (internal/curriculum/store.Seed* at the snapshot commit).
type v1Phase struct {
	PathSlug string `json:"path_slug"`
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"week_from"`
	WeekTo   int    `json:"week_to"`
}

type v1Week struct {
	PathSlug string `json:"path_slug"`
	N        int    `json:"n"`
	Title    string `json:"title"`
	Thesis   string `json:"thesis"`
}

type v1Concept struct {
	PathSlug     string `json:"path_slug"`
	Slug         string `json:"slug"`
	Title        string `json:"title"`
	BodyMD       string `json:"body_md"`
	WhenToUseMD  string `json:"when_to_use_md"`
	CodeTemplate string `json:"code_template"`
	Weeks        []int  `json:"weeks"`
}

type v1Problem struct {
	ID              string      `json:"id"`
	PathSlug        string      `json:"path_slug"`
	WeekN           int         `json:"week_n"`
	Title           string      `json:"title"`
	Difficulty      string      `json:"difficulty"`
	Pattern         string      `json:"pattern"`
	LeetcodeURL     string      `json:"leetcode_url"`
	NeetcodeURL     string      `json:"neetcode_url"`
	IsReinforcement bool        `json:"is_reinforcement"`
	SortOrder       int         `json:"sort_order"`
	Sections        []v1Section `json:"sections"`
}

type v1Section struct {
	Stage  string `json:"stage"`
	Kind   string `json:"kind"`
	Order  int    `json:"order"`
	BodyMD string `json:"body_md"`
	Code   string `json:"code"`
}

// Output rows: path_slug is dropped (the course directory is the course).
type phaseRow struct {
	Order    int    `json:"order"`
	Name     string `json:"name"`
	Theme    string `json:"theme"`
	WeekFrom int    `json:"week_from"`
	WeekTo   int    `json:"week_to"`
}

type weekRow struct {
	N      int    `json:"n"`
	Title  string `json:"title"`
	Thesis string `json:"thesis"`
}

type conceptRow struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Weeks []int  `json:"weeks"`
}

type lockEntry struct {
	Course string `json:"course"`
	Status string `json:"status"`
}

type idsLock struct {
	Items  map[string]lockEntry `json:"items"`
	Assets map[string]string    `json:"assets"`
}

const dsaSlug = course.DSASlug

// copiedExample1 are the two items whose attempt-stage Example 1 v1 copied from
// LeetCode (t1 §8). They convert as "adapted" until the Example-1 commit rewrites them
// and flips them to "original".
var copiedExample1 = map[string]bool{"3": true, "16": true}

var (
	stages = map[string]bool{"attempt": true, "hint": true, "solution": true}
	kindRe = regexp.MustCompile(`^[a-z][a-z_]*$`)
	slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
)

func main() {
	root := flag.String("root", "curriculum", "the curriculum/ directory")
	flag.Parse()
	if err := convert(*root); err != nil {
		log.Fatal(err)
	}
}

func convert(root string) error {
	var (
		phases   []v1Phase
		weeks    []v1Week
		concepts []v1Concept
		problems []v1Problem
	)
	for _, f := range []struct {
		name string
		v    any
	}{
		{"dsa/phases.json", &phases}, {"dsa/weeks.json", &weeks},
		{"dsa/concepts.json", &concepts}, {"dsa/problems.json", &problems},
	} {
		if err := readStrict(filepath.Join(root, f.name), f.v); err != nil {
			return err
		}
	}

	dir := filepath.Join(root, "courses", dsaSlug)
	// One-shot but re-runnable: clear everything but the m1-01 manifest.
	for _, p := range []string{"phases.json", "weeks.json", "concepts.json", "concepts", "items"} {
		if err := os.RemoveAll(filepath.Join(dir, p)); err != nil {
			return err
		}
	}

	var outPhases []phaseRow
	for _, p := range phases {
		if p.PathSlug != dsaSlug {
			return fmt.Errorf("phase %d: path_slug %q is not %q", p.Order, p.PathSlug, dsaSlug)
		}
		outPhases = append(outPhases, phaseRow{p.Order, p.Name, p.Theme, p.WeekFrom, p.WeekTo})
	}
	if err := writeJSON(filepath.Join(dir, "phases.json"), outPhases); err != nil {
		return err
	}

	var outWeeks []weekRow
	for _, w := range weeks {
		if w.PathSlug != dsaSlug {
			return fmt.Errorf("week %d: path_slug %q is not %q", w.N, w.PathSlug, dsaSlug)
		}
		outWeeks = append(outWeeks, weekRow{w.N, w.Title, w.Thesis})
	}
	if err := writeJSON(filepath.Join(dir, "weeks.json"), outWeeks); err != nil {
		return err
	}

	var outConcepts []conceptRow
	for _, c := range concepts {
		if c.PathSlug != dsaSlug {
			return fmt.Errorf("concept %q: path_slug %q is not %q", c.Slug, c.PathSlug, dsaSlug)
		}
		if !slugRe.MatchString(c.Slug) {
			return fmt.Errorf("concept slug %q is not a slug", c.Slug)
		}
		outConcepts = append(outConcepts, conceptRow{c.Slug, c.Title, c.Weeks})
		for _, sc := range []struct{ rel, body string }{
			{filepath.Join("concepts", c.Slug+".md"), c.BodyMD},
			{filepath.Join("concepts", c.Slug+".when.md"), c.WhenToUseMD},
			{filepath.Join("concepts", "_code", c.Slug+".go.snip"), c.CodeTemplate},
		} {
			if sc.body == "" {
				continue // an absent sidecar is the empty string
			}
			if err := writeRaw(filepath.Join(dir, sc.rel), sc.body); err != nil {
				return err
			}
		}
	}
	if err := writeJSON(filepath.Join(dir, "concepts.json"), outConcepts); err != nil {
		return err
	}

	lock := idsLock{Items: map[string]lockEntry{}, Assets: map[string]string{}}
	for _, p := range problems {
		if p.PathSlug != dsaSlug {
			return fmt.Errorf("problem %s: path_slug %q is not %q", p.ID, p.PathSlug, dsaSlug)
		}
		if _, dup := lock.Items[p.ID]; dup {
			return fmt.Errorf("problem %s: duplicate id", p.ID)
		}
		it, err := toItem(p)
		if err != nil {
			return err
		}
		idir := filepath.Join(dir, "items", p.ID)
		if err := writeJSON(filepath.Join(idir, "item.json"), it); err != nil {
			return err
		}
		if err := writeSections(idir, p); err != nil {
			return err
		}
		lock.Items[p.ID] = lockEntry{Course: dsaSlug, Status: it.Status}
	}
	return writeJSON(filepath.Join(root, "ids.lock.json"), lock)
}

// toItem maps a v1 problem to an item.json (frozen schema v1): role from
// is_reinforcement, links from the urls, provenance from the LeetCode link and the git
// history (the v1 seed was written in build session S03 with an AI assistant).
func toItem(p v1Problem) (*course.Item, error) {
	role := "core"
	if p.IsReinforcement {
		role = "reinforcement"
	}
	origin := "original"
	if copiedExample1[p.ID] {
		origin = "adapted"
	}
	it := &course.Item{
		ID:         p.ID,
		Course:     dsaSlug,
		WeekN:      p.WeekN,
		SortOrder:  p.SortOrder,
		Title:      p.Title,
		Difficulty: p.Difficulty,
		Pattern:    p.Pattern,
		Role:       role,
		Status:     "live",
		Provenance: course.Provenance{Origin: origin, AuthoredBy: "ai-assisted"},
	}
	if p.LeetcodeURL != "" {
		ref, err := leetcodeRef(p.LeetcodeURL)
		if err != nil {
			return nil, fmt.Errorf("problem %s: %w", p.ID, err)
		}
		it.Provenance.InspiredBy = []string{ref}
		it.Links = append(it.Links, course.Link{Kind: "leetcode", URL: p.LeetcodeURL})
	}
	if p.NeetcodeURL != "" {
		it.Links = append(it.Links, course.Link{Kind: "neetcode", URL: p.NeetcodeURL})
	}
	if err := it.Validate(); err != nil {
		return nil, fmt.Errorf("problem %s: converted item is invalid: %w", p.ID, err)
	}
	return it, nil
}

// leetcodeRef turns https://leetcode.com/problems/<slug>[/] into leetcode:<slug>.
func leetcodeRef(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	slug, ok := strings.CutPrefix(strings.Trim(u.Path, "/"), "problems/")
	if u.Host != "leetcode.com" || !ok || !slugRe.MatchString(slug) {
		return "", fmt.Errorf("unexpected LeetCode URL %q", raw)
	}
	return "leetcode:" + slug, nil
}

// writeSections writes prose sections as sections/<stage>/<NN>-<kind>.md and code
// sections as _code/<stage>-<NN>.go.snip (language = the extension before .snip).
func writeSections(idir string, p v1Problem) error {
	seen := map[string]bool{}
	secs := append([]v1Section(nil), p.Sections...)
	sort.SliceStable(secs, func(i, j int) bool {
		if secs[i].Stage != secs[j].Stage {
			return secs[i].Stage < secs[j].Stage
		}
		return secs[i].Order < secs[j].Order
	})
	for _, s := range secs {
		if !stages[s.Stage] {
			return fmt.Errorf("problem %s: unknown stage %q", p.ID, s.Stage)
		}
		if !kindRe.MatchString(s.Kind) {
			return fmt.Errorf("problem %s: kind %q is not a snake_case name", p.ID, s.Kind)
		}
		if s.Order < 1 || s.Order > 99 {
			return fmt.Errorf("problem %s: order %d is outside 1..99", p.ID, s.Order)
		}
		key := fmt.Sprintf("%s/%d", s.Stage, s.Order)
		if seen[key] {
			return fmt.Errorf("problem %s: duplicate section %s", p.ID, key)
		}
		seen[key] = true
		var rel, body string
		switch s.Kind {
		case "code":
			if s.BodyMD != "" {
				return fmt.Errorf("problem %s: code section %s carries prose", p.ID, key)
			}
			rel = filepath.Join("_code", fmt.Sprintf("%s-%02d.go.snip", s.Stage, s.Order))
			body = s.Code
		default:
			if s.Code != "" {
				return fmt.Errorf("problem %s: prose section %s carries code", p.ID, key)
			}
			rel = filepath.Join("sections", s.Stage, fmt.Sprintf("%02d-%s.md", s.Order, s.Kind))
			body = s.BodyMD
		}
		if body == "" {
			return fmt.Errorf("problem %s: empty section %s would vanish (absent sidecar)", p.ID, key)
		}
		if err := writeRaw(filepath.Join(idir, rel), body); err != nil {
			return err
		}
	}
	return nil
}

func readStrict(name string, v any) error {
	b, err := os.ReadFile(name)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	if dec.More() {
		return errors.New("trailing data in " + name)
	}
	return nil
}

// writeJSON writes v indented (2 spaces), unescaped, with one trailing newline.
func writeJSON(name string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return writeRaw(name, buf.String())
}

// writeRaw writes body byte-exact.
func writeRaw(name, body string) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, []byte(body), 0o644)
}
