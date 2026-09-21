package gateway

import (
	"os"
	"sort"
	"strings"
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// specPath is the committed OpenAPI contract, relative to this package directory.
const specPath = "../../docs/architecture/openapi.yaml"

// TestOpenAPISpecMatchesRoutes is the CI drift check (Task 3): it fails when the
// committed OpenAPI spec and the gateway's route table diverge — a route added to
// apiRoutes without a matching spec entry, or a spec path with no route. This keeps
// docs/architecture/openapi.yaml honest as the handler surface evolves (ADR-0021).
// It is a pure Go test (no network/DB), so it runs in the normal `go test ./...` lane.
func TestOpenAPISpecMatchesRoutes(t *testing.T) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read openapi spec: %v", err)
	}
	var doc struct {
		OpenAPI string                            `yaml:"openapi"`
		Paths   map[string]map[string]interface{} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse openapi spec: %v", err)
	}
	if !strings.HasPrefix(doc.OpenAPI, "3.1") {
		t.Errorf("openapi version = %q, want 3.1.x", doc.OpenAPI)
	}
	if len(doc.Paths) == 0 {
		t.Fatal("openapi spec has no paths")
	}

	httpMethods := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}

	// Set from the spec: "METHOD /path".
	spec := map[string]bool{}
	for path, ops := range doc.Paths {
		for method := range ops {
			if httpMethods[strings.ToLower(method)] {
				spec[strings.ToUpper(method)+" "+path] = true
			}
		}
	}

	// Set from the authoritative route table: documented /api routes with the /api
	// prefix stripped (the OpenAPI server URL is /xlearn/api, so paths are relative).
	reg := map[string]bool{}
	for _, rt := range (&Gateway{}).apiRoutes() {
		if !rt.Doc {
			continue // ops/JWKS routes are intentionally outside the documented surface
		}
		reg[rt.Method+" "+strings.TrimPrefix(rt.Pattern, "/api")] = true
	}

	for _, k := range diff(reg, spec) {
		t.Errorf("route %q is not documented in openapi.yaml (add it to the spec)", k)
	}
	for _, k := range diff(spec, reg) {
		t.Errorf("openapi.yaml documents %q which is not a gateway route (spec is stale)", k)
	}
}

// diff returns the sorted keys present in a but not in b.
func diff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
