package config

import "testing"

func TestNormalizeBasePath(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"/":            "",
		"/xlearn":      "/xlearn",
		"/xlearn/":     "/xlearn",
		"xlearn":       "/xlearn",
		"  /xlearn/  ": "/xlearn",
		"/xlearn/api/": "/xlearn/api",
	}
	for in, want := range cases {
		if got := normalizeBasePath(in); got != want {
			t.Errorf("normalizeBasePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"PORT", "LOG_LEVEL", "BASE_PATH"} {
		t.Setenv(k, "")
	}
	c := Load()
	if c.Port != defaultPort || c.LogLevel != defaultLogLevel || c.BasePath != defaultBasePath {
		t.Fatalf("Load() defaults = %+v", c)
	}
	if c.Addr() != ":8080" {
		t.Fatalf("Addr() = %q, want :8080", c.Addr())
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("BASE_PATH", "/app/")
	c := Load()
	if c.Port != "9090" || c.LogLevel != "debug" || c.BasePath != "/app" {
		t.Fatalf("Load() = %+v", c)
	}
}
