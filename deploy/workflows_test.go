package deploy

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The release streams must never collide (ADR-0034 §1.5; docs/git-strategy.md "Runner stream"):
// a runner tag (runner-v*) runs runner-release.yml and nothing that builds the fleet, and a fleet
// tag (v*) never runs runner-release.yml. This test reads every workflow's `on:` block and applies
// GitHub's filter-pattern rules to a tag push. The live proof is the rc rehearsal (m3-15 task 8:
// no deploy.yml run for the runner-v1.0.0-rc.1 ref); deploy.yml's own `^v…` guard is the backstop.

func TestTagTriggersNeverCollide(t *testing.T) {
	runnerTags := []string{"runner-v1.0.0", "runner-v1.0.0-rc.1", "runner-v2.3.4"}
	fleetTags := []string{"v1.13.0", "v2.0.0-rc.1", "v1.7.0"}
	triggered := func(wf, tag string) bool {
		on := workflowOn(t, wf)
		return tagTriggers(t, on, tag)
	}
	for _, tag := range runnerTags {
		if triggered("deploy.yml", tag) {
			t.Errorf("deploy.yml runs for the runner tag %s: the fleet pipeline would build the runner", tag)
		}
		if !triggered("runner-release.yml", tag) {
			t.Errorf("runner-release.yml does not run for %s", tag)
		}
	}
	for _, tag := range fleetTags {
		if triggered("runner-release.yml", tag) {
			t.Errorf("runner-release.yml runs for the fleet tag %s", tag)
		}
		if !triggered("deploy.yml", tag) {
			t.Errorf("deploy.yml does not run for the fleet tag %s", tag)
		}
	}
	// No other workflow is started by a release tag of either stream.
	files, _ := filepath.Glob("../.github/workflows/*.yml")
	for _, f := range files {
		wf := filepath.Base(f)
		if wf == "deploy.yml" || wf == "runner-release.yml" {
			continue
		}
		for _, tag := range append(append([]string{}, runnerTags...), fleetTags...) {
			if triggered(wf, tag) {
				t.Errorf("%s runs for the tag %s", wf, tag)
			}
		}
	}
}

// TestGlob pins the filter-pattern semantics the collision test relies on.
func TestGlob(t *testing.T) {
	for _, c := range []struct {
		pat, s string
		want   bool
	}{
		{"v*", "v1.0.0", true},
		{"v*", "runner-v1.0.0", false},
		{"runner-v*", "runner-v1.0.0-rc.1", true},
		{"runner-v*", "v1.0.0", false},
		{"v*", "v1/x", false},
		{"v**", "v1/x", true},
		{"v[12].*", "v2.0", true},
		{"v[12].*", "v3.0", false},
		{"v1.?0", "v1.0", true},
		{"v1+.0", "v111.0", true},
	} {
		if got := globMatch(c.pat, c.s); got != c.want {
			t.Errorf("glob %q on %q = %v, want %v", c.pat, c.s, got, c.want)
		}
	}
}

// TestOneBuildKitPin: CI's runner-repro and runner-image-acceptance, runner-release.yml and the
// Dockerfile's local-rebuild recipe name the same pinned BuildKit image, so a local rebuild can
// reproduce the pushed digest (layer bytes depend on the BuildKit version and its compression).
func TestOneBuildKitPin(t *testing.T) {
	re := regexp.MustCompile(`moby/buildkit:v[0-9.]+@sha256:[0-9a-f]{64}`)
	seen := map[string][]string{}
	for _, f := range []string{"../.github/workflows/runner-release.yml", "../.github/workflows/ci.yml", "runner.Dockerfile"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		pins := re.FindAllString(string(b), -1)
		if len(pins) == 0 {
			t.Errorf("%s names no pinned moby/buildkit image", f)
		}
		for _, p := range pins {
			seen[p] = append(seen[p], f)
		}
	}
	if len(seen) > 1 {
		t.Errorf("more than one BuildKit pin: %v", seen)
	}
}

// TestRunnerReleaseLine: deploy/runner.release-line holds the runner's live major (1 until a
// judge↔runner contract break moves it in a reviewed PR); the fleet's .release-line is separate.
func TestRunnerReleaseLine(t *testing.T) {
	b, err := os.ReadFile("runner.release-line")
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9]+$`).MatchString(strings.TrimSpace(string(b))) {
		t.Errorf("deploy/runner.release-line = %q, want a major version", b)
	}
}

// ---- the `on:` block ----

func workflowOn(t *testing.T, name string) any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(b, &doc); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	// yaml.v3 decodes the key `on` as the string "on" (YAML 1.2), never as the boolean true.
	on, ok := doc["on"]
	if !ok {
		t.Fatalf("%s has no on: block", name)
	}
	return on
}

// tagTriggers reports whether a push of tag starts a workflow with this `on:` value (GitHub's
// rules: a bare `push` runs for every tag; a push filter with branches but no tags runs for no
// tag; tags / tags-ignore filter the tag name).
func tagTriggers(t *testing.T, on any, tag string) bool {
	t.Helper()
	switch v := on.(type) {
	case string:
		return v == "push"
	case []any:
		for _, e := range v {
			if e == "push" {
				return true
			}
		}
		return false
	case map[string]any:
		push, ok := v["push"]
		if !ok {
			return false
		}
		m, ok := push.(map[string]any)
		if !ok || m == nil {
			return true // `push:` with no filters
		}
		tags, hasTags := m["tags"]
		ignore, hasIgnore := m["tags-ignore"]
		switch {
		case hasTags:
			return filterMatch(strings_(t, tags), tag)
		case hasIgnore:
			return !filterMatch(strings_(t, ignore), tag)
		case m["branches"] != nil || m["branches-ignore"] != nil || m["paths"] != nil || m["paths-ignore"] != nil:
			return false
		}
		return true
	}
	t.Fatalf("unexpected on: %T", on)
	return false
}

func strings_(t *testing.T, v any) []string {
	t.Helper()
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		out := make([]string, len(x))
		for i, e := range x {
			s, ok := e.(string)
			if !ok {
				t.Fatalf("a non-string filter pattern: %v", e)
			}
			out[i] = s
		}
		return out
	}
	t.Fatalf("unexpected filter %T", v)
	return nil
}

// filterMatch applies a pattern list in order: a match includes, a later `!pattern` excludes.
func filterMatch(pats []string, s string) bool {
	in := false
	for _, p := range pats {
		if neg, ok := strings.CutPrefix(p, "!"); ok {
			if globMatch(neg, s) {
				in = false
			}
		} else if globMatch(p, s) {
			in = true
		}
	}
	return in
}

// globMatch is GitHub's filter-pattern syntax: `*` any run without `/`, `**` any run, `?` zero or
// one of the preceding character, `+` one or more of it, `[…]` a class.
func globMatch(pat, s string) bool {
	var re strings.Builder
	re.WriteString("^")
	for i := 0; i < len(pat); i++ {
		switch c := pat[i]; c {
		case '*':
			if i+1 < len(pat) && pat[i+1] == '*' {
				re.WriteString(".*")
				i++
			} else {
				re.WriteString("[^/]*")
			}
		case '?', '+':
			re.WriteByte(c)
		case '[':
			j := strings.IndexByte(pat[i:], ']')
			if j < 0 {
				re.WriteString(regexp.QuoteMeta(pat[i:]))
				i = len(pat)
				continue
			}
			re.WriteString(pat[i : i+j+1])
			i += j
		default:
			re.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	re.WriteString("$")
	ok, err := regexp.MatchString(re.String(), s)
	return err == nil && ok
}
