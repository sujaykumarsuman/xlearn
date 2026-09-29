package admin

import (
	"bytes"
	"context"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// Bad arguments exit 2 without ever connecting to the database.
func TestUsageErrorsNeverConnect(t *testing.T) {
	deps := Deps{Open: func(context.Context) (Store, func(), error) {
		t.Fatal("connected on a usage error")
		return nil, nil, nil
	}}
	for _, args := range [][]string{
		{},
		{"nope"},
		{"seats", "extra"},
		{"account"},
		{"account", "erase", "x"},
		{"account", "suspend"},
		{"account", "suspend", "a", "b"},
		{"account", "suspend", "--force"},
		{"account", "set-role", "a"},
		{"account", "set-role", "a", "admin"},
		{"account", "list", "--role", "admin"},
		{"account", "list", "--status", "gone"},
		{"account", "list", "--dormant", "soon"},
		{"account", "list", "stray"},
		{"account", "create", "--email", "t@example.com"},
		{"account", "create", "--role", "learner", "--email", "t@example.com"},
		{"account", "create", "--role", "owner", "--email", "t@example.com"},
		{"account", "create", "--role", "tester", "--email", "not-an-email"},
		{"account", "create", "--role", "tester"},
	} {
		var out, errb bytes.Buffer
		if code := Run(context.Background(), deps, args, &out, &errb); code != ExitUsage {
			t.Errorf("%q: exit %d, want %d (stderr %s)", args, code, ExitUsage, errb.String())
		}
		if out.Len() != 0 {
			t.Errorf("%q: wrote to stdout: %s", args, out.String())
		}
	}
	var errb bytes.Buffer
	if code := Run(context.Background(), deps, []string{"help"}, &bytes.Buffer{}, &errb); code != ExitOK || !strings.Contains(errb.String(), "account set-role") {
		t.Fatalf("help: exit %d %s", code, errb.String())
	}
}

func TestParseWindow(t *testing.T) {
	for in, want := range map[string]time.Duration{"30d": 30 * 24 * time.Hour, "1d": 24 * time.Hour, "72h": 72 * time.Hour, "90m": 90 * time.Minute} {
		if got, err := parseWindow(in); err != nil || got != want {
			t.Errorf("parseWindow(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "0d", "-3d", "d", "soon", "-1h", "0s"} {
		if _, err := parseWindow(in); err == nil {
			t.Errorf("parseWindow(%q) accepted", in)
		}
	}
}

func TestNewPassword(t *testing.T) {
	seen := map[string]bool{}
	for range 50 {
		pw, err := NewPassword()
		if err != nil {
			t.Fatal(err)
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{24}$`).MatchString(pw) {
			t.Fatalf("password %q is not 24 base64url characters", pw)
		}
		if seen[pw] {
			t.Fatal("repeated password")
		}
		seen[pw] = true
	}
}

func TestExitCodes(t *testing.T) {
	for err, want := range map[error]int{
		nil:                                ExitOK,
		store.ErrNotFound:                  ExitNotFound,
		store.ErrLastOwner:                 ExitRefused,
		store.ErrSeatCap:                   ExitRefused,
		store.ErrEmailTaken:                ExitRefused,
		errors.New("connection refused"):   ExitError,
		errors.Join(store.ErrSeatCap, nil): ExitRefused,
	} {
		if got, _ := exitFor(err); got != want {
			t.Errorf("exitFor(%v) = %d, want %d", err, got, want)
		}
	}
}
