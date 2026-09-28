package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nats-io/nkeys"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

// writeKeys writes a public-key file for every identity (throwaway keys).
func writeKeys(t *testing.T) (path string, pubs map[string]string) {
	t.Helper()
	pubs = map[string]string{}
	var b strings.Builder
	b.WriteString("# public keys only\n\n")
	for _, id := range events.ACLIdentities() {
		kp, err := nkeys.CreateUser()
		if err != nil {
			t.Fatal(err)
		}
		pk, _ := kp.PublicKey()
		pubs[id] = pk
		b.WriteString(id + "=" + pk + "\n")
	}
	path = filepath.Join(t.TempDir(), "pubkeys.env")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return path, pubs
}

func TestRenderAllStagesAndFormats(t *testing.T) {
	path, pubs := writeKeys(t)
	for _, legacy := range []string{"allow", "deny", "none"} {
		for _, format := range []string{"conf", "yaml"} {
			var out bytes.Buffer
			if err := run([]string{"-nkeys", path, "-legacy", legacy, "-format", format}, &out); err != nil {
				t.Fatalf("%s/%s: %v", legacy, format, err)
			}
			s := out.String()
			if !strings.Contains(s, pubs["practice"]) || !strings.Contains(s, pubs["ops"]) {
				t.Errorf("%s/%s: real public keys not rendered", legacy, format)
			}
			if strings.Contains(s, "<NKEY_PUB:") {
				t.Errorf("%s/%s: placeholder key left in a real render", legacy, format)
			}
			hasLegacy := strings.Contains(s, `"legacy"`)
			if (legacy == "none") == hasLegacy {
				t.Errorf("%s/%s: legacy user present = %v", legacy, format, hasLegacy)
			}
			if legacy != "none" && strings.Contains(s, events.LegacyPasswordPlaceholder) {
				t.Errorf("%s/%s: the legacy password must be randomised", legacy, format)
			}
		}
	}
}

func TestRejectsSeedsAndBadInput(t *testing.T) {
	kp, _ := nkeys.CreateUser()
	seed, _ := kp.Seed()
	dir := t.TempDir()
	bad := filepath.Join(dir, "seed.env")
	if err := os.WriteFile(bad, []byte("practice="+string(seed)+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-nkeys", bad}, &out); err == nil || !strings.Contains(err.Error(), "SEED") {
		t.Fatalf("a seed in the key file must be refused, got %v", err)
	}
	if out.Len() != 0 {
		t.Fatal("nothing may be printed when a seed is passed")
	}
	path, _ := writeKeys(t)
	for _, args := range [][]string{
		{},
		{"-nkeys", path, "-legacy", "sometimes"},
		{"-nkeys", path, "-format", "json"},
		{"-nkeys", filepath.Join(dir, "absent")},
	} {
		if err := run(args, &bytes.Buffer{}); err == nil {
			t.Errorf("run(%v) must fail", args)
		}
	}
}
