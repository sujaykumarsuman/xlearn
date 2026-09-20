package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testGateway(basePath string) *Gateway {
	dist := fstest.MapFS{
		"index.html":              {Data: []byte("<!doctype html><title>xLearn</title>")},
		"assets/index-abc123.js":  {Data: []byte("console.log(1)")},
		"assets/index-abc123.css": {Data: []byte("body{}")},
		"favicon.svg":             {Data: []byte("<svg/>")},
	}
	return New(Options{BasePath: basePath, Version: "test", Dist: dist})
}

func do(g *Gateway, method, target string) *http.Response {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	g.Handler().ServeHTTP(w, r)
	return w.Result()
}

func TestK8sProbes(t *testing.T) {
	g := testGateway("/xlearn")
	for _, p := range []string{"/healthz", "/readyz"} {
		if resp := do(g, http.MethodGet, p); resp.StatusCode != http.StatusOK {
			t.Errorf("%s = %d, want 200", p, resp.StatusCode)
		}
	}
}

func TestAppHealth_WithAndWithoutPrefix(t *testing.T) {
	g := testGateway("/xlearn")
	// Local access (un-stripped) and behind-Traefik access (stripped) both work.
	for _, p := range []string{"/xlearn/api/healthz", "/api/healthz"} {
		resp := do(g, http.MethodGet, p)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s = %d, want 200", p, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), `"service":"gateway"`) {
			t.Errorf("%s body = %q", p, body)
		}
	}
}

func TestUnknownAPI404(t *testing.T) {
	g := testGateway("/xlearn")
	if resp := do(g, http.MethodGet, "/xlearn/api/does-not-exist"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown /api/* = %d, want 404", resp.StatusCode)
	}
}

func TestHashedAssetImmutable(t *testing.T) {
	g := testGateway("/xlearn")
	resp := do(g, http.MethodGet, "/xlearn/assets/index-abc123.js")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("asset = %d, want 200", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Errorf("asset Cache-Control = %q, want immutable", cc)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/javascript") && !strings.Contains(ct, "javascript") {
		t.Errorf("asset Content-Type = %q, want javascript", ct)
	}
}

func TestSPAFallback(t *testing.T) {
	g := testGateway("/xlearn")
	// A deep client-side route falls back to index.html (200, no-cache).
	resp := do(g, http.MethodGet, "/xlearn/dsa/week/2")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("deep link = %d, want 200", resp.StatusCode)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("index Cache-Control = %q, want no-cache", cc)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "xLearn") {
		t.Errorf("fallback did not serve index.html: %q", body)
	}
}

func TestRootServesIndex(t *testing.T) {
	g := testGateway("/xlearn")
	for _, p := range []string{"/xlearn", "/xlearn/", "/"} {
		resp := do(g, http.MethodGet, p)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s = %d, want 200", p, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "xLearn") {
			t.Errorf("%s did not serve index.html", p)
		}
	}
}

func TestRootMountedGateway(t *testing.T) {
	// BasePath "" (root mount): probes and SPA still resolve.
	g := testGateway("")
	if resp := do(g, http.MethodGet, "/api/healthz"); resp.StatusCode != http.StatusOK {
		t.Errorf("/api/healthz (root mount) = %d, want 200", resp.StatusCode)
	}
	if resp := do(g, http.MethodGet, "/dsa/dashboard"); resp.StatusCode != http.StatusOK {
		t.Errorf("SPA fallback (root mount) = %d, want 200", resp.StatusCode)
	}
}
