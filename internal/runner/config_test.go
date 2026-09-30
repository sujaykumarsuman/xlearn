package runner

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, k := range []string{"RUNNER_ADDR", "RUNNER_TOKEN_FILE", "RUNNER_MODE", "RUNNER_IMAGE_DIGEST", "RUNNER_MAX_JOBS",
		"RUNNER_MAX_AGE", "RUNNER_SPAWN_PATH", "RUNNER_DRAIN_TIMEOUT"} {
		t.Setenv(k, "")
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8090" || c.TokenFile != "/var/run/secrets/runner-auth/token" || c.Mode != ModeProd ||
		c.ImageDigest != "unknown" || c.MaxJobs != 500 || c.MaxAge != 6*time.Hour || c.SpawnPath != SpawnCloneIntoCgroup ||
		c.DrainTimeout != 60*time.Second || c.FrontUID != 65532 {
		t.Errorf("defaults: %+v", c)
	}
}

func TestLoadRejects(t *testing.T) {
	for k, v := range map[string]string{
		"RUNNER_MODE":          "staging",
		"RUNNER_SPAWN_PATH":    "ptrace",
		"RUNNER_MAX_JOBS":      "0",
		"RUNNER_MAX_AGE":       "10s",
		"RUNNER_DRAIN_TIMEOUT": "90s",
		"RUNNER_FRONT_UID":     "0",
	} {
		t.Run(k, func(t *testing.T) {
			t.Setenv(k, v)
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), k) {
				t.Errorf("%s=%s: want an error naming it, got %v", k, v, err)
			}
		})
	}
}

func TestReadManifest(t *testing.T) {
	m, err := ReadManifest(strings.NewReader(`{"token":"t0123456789abcdef","boot_epoch":"h/1/00","profiles":[{"name":"go@1.26"}]}`))
	if err != nil || m.Profiles[0].Name != "go@1.26" {
		t.Fatalf("%+v %v", m, err)
	}
	if _, err := ReadManifest(strings.NewReader(`{"boot_epoch":"x"}`)); err == nil {
		t.Error("a manifest without a token must be refused")
	}
	if _, err := ReadManifest(strings.NewReader(strings.Repeat(" ", MaxManifestBytes+1))); err == nil {
		t.Error("an oversize manifest must be refused")
	}
}
