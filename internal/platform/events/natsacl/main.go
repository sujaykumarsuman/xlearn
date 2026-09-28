// Command natsacl prints the NATS `authorization` block rendered from
// internal/platform/events/topology.go (ADR-0035 §2) with real public nkeys, for
// mi-06's N1/N3/N4 infra PRs:
//
//	make nats-acl-render NKEYS=<pubkeys.env> LEGACY=allow|deny|none FORMAT=conf|yaml
//
// The NKEYS file holds `<identity>=U…` lines — PUBLIC keys only, one per service plus
// ops (blank lines and # comments are ignored). Never pass a seed. With LEGACY=allow or
// deny the legacy user gets a fresh random password that no client ever uses
// (anonymous clients reach it through no_auth_user).
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "natsacl:", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("natsacl", flag.ContinueOnError)
	nkeysPath := fs.String("nkeys", "", "file of <identity>=<public nkey> lines (public keys only)")
	legacy := fs.String("legacy", string(events.LegacyAllow), "legacy bridge stage: allow (N1), deny (N3) or none (N4)")
	format := fs.String("format", "yaml", "output form: conf (NATS server conf) or yaml (nats chart values fragment)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *nkeysPath == "" {
		return fmt.Errorf("-nkeys is required")
	}
	mode, err := events.ParseLegacyMode(*legacy)
	if err != nil {
		return err
	}
	keys, err := readKeys(*nkeysPath)
	if err != nil {
		return err
	}
	a, err := events.RenderAuthorization(keys, mode)
	if err != nil {
		return err
	}
	if mode != events.LegacyNone {
		pw, err := randomPassword()
		if err != nil {
			return err
		}
		a.SetLegacyPassword(pw)
	}
	switch *format {
	case "conf":
		_, err = io.WriteString(out, a.Conf())
	case "yaml":
		_, err = io.WriteString(out, a.ChartValues())
	default:
		return fmt.Errorf("-format must be conf or yaml, got %q", *format)
	}
	return err
}

// readKeys parses `<identity>=<public key>` lines. A value that looks like a seed
// (S…) is rejected outright so a seed never reaches a rendered file.
func readKeys(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	keys := map[string]string{}
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		id, key, ok := strings.Cut(line, "=")
		id, key = strings.TrimSpace(id), strings.TrimSpace(key)
		if !ok || id == "" || key == "" {
			return nil, fmt.Errorf("%s:%d: want <identity>=<public nkey>", path, n)
		}
		if strings.HasPrefix(key, "S") {
			return nil, fmt.Errorf("%s:%d: %s looks like a SEED; pass public keys only", path, n, id)
		}
		if _, dup := keys[id]; dup {
			return nil, fmt.Errorf("%s:%d: duplicate identity %q", path, n, id)
		}
		keys[id] = key
	}
	return keys, sc.Err()
}

func randomPassword() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
