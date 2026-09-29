package identity

import (
	"bytes"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDSNEncodesReservedCharacters(t *testing.T) {
	// Passwords a random base64/secret generator can emit; each contains a byte
	// that would corrupt a raw-concatenated postgres:// URL.
	passwords := []string{
		"Ab3/x9+k=", // '/' truncates the authority
		"p@ssw0rd",  // '@' splits userinfo/host
		"a b c",     // whitespace
		"a:b:c",     // extra colons
		"plainhex0123456789abcdef",
	}
	for _, pw := range passwords {
		db := DBConfig{
			Host:     "projects-pgstore-rw.databases.svc.cluster.local",
			Port:     "5432",
			Database: "xlearndb",
			User:     "xlearn_identity",
			Password: pw,
			SSLMode:  "prefer",
		}
		cfg, err := pgxpool.ParseConfig(db.DSN())
		if err != nil {
			t.Fatalf("ParseConfig(pw=%q) failed: %v (dsn=%q)", pw, err, db.DSN())
		}
		if cfg.ConnConfig.User != db.User {
			t.Errorf("pw=%q: user = %q, want %q", pw, cfg.ConnConfig.User, db.User)
		}
		if cfg.ConnConfig.Password != pw {
			t.Errorf("pw=%q: password round-trip = %q", pw, cfg.ConnConfig.Password)
		}
		if cfg.ConnConfig.Host != db.Host {
			t.Errorf("pw=%q: host = %q, want %q", pw, cfg.ConnConfig.Host, db.Host)
		}
		if cfg.ConnConfig.Database != db.Database {
			t.Errorf("pw=%q: database = %q, want %q", pw, cfg.ConnConfig.Database, db.Database)
		}
	}
}

// L7 (ADR-0033 §3, ADR-0035 §4): `open` is honoured only with DEV_AUTH; `invite` runs
// closed until L-A; anything else is closed. The full SIGNUP_MODE × DEV_AUTH table goes
// through LoadConfig (the real env parsing), and the note decides the log level.
func TestSignupModeL7Guard(t *testing.T) {
	const unset = "\x00unset"
	modes := []string{unset, "", "open", "OPEN", " open ", "closed", "invite", "junk"}
	devAuths := []string{unset, "true", "false", "junk"}
	for _, mode := range modes {
		for _, dev := range devAuths {
			setOrUnset(t, "SIGNUP_MODE", mode, unset)
			setOrUnset(t, "DEV_AUTH", dev, unset)
			a := LoadConfig().Auth

			devOn := dev == "true" // "junk" and "false" parse as off (fail safe)
			if a.DevAuth != devOn {
				t.Fatalf("DEV_AUTH=%q: DevAuth=%v, want %v", dev, a.DevAuth, devOn)
			}
			isOpen := mode == "open" || mode == "OPEN" || mode == " open "
			want := SignupClosed
			if isOpen && devOn {
				want = SignupOpen
			}
			if a.Signup != want {
				t.Errorf("SIGNUP_MODE=%q DEV_AUTH=%q: Signup=%q, want %q", mode, dev, a.Signup, want)
			}
			// The note: the L7 misconfiguration and invite/junk say why; open+DEV_AUTH,
			// closed and unset/empty say nothing.
			wantNote := (isOpen && !devOn) || mode == "invite" || mode == "junk"
			if (a.SignupNote != "") != wantNote {
				t.Errorf("SIGNUP_MODE=%q DEV_AUTH=%q: note %q, want note=%v", mode, dev, a.SignupNote, wantNote)
			}
			level := logLevelOf(a)
			switch {
			case isOpen && !devOn:
				if level != "ERROR" || !strings.Contains(a.SignupNote, "ignored without DEV_AUTH") {
					t.Errorf("SIGNUP_MODE=%q DEV_AUTH=%q: logged %q %q, want ERROR 'ignored without DEV_AUTH'", mode, dev, level, a.SignupNote)
				}
			case wantNote:
				if level != "INFO" {
					t.Errorf("SIGNUP_MODE=%q DEV_AUTH=%q: logged at %q, want INFO", mode, dev, level)
				}
			default:
				if level != "" {
					t.Errorf("SIGNUP_MODE=%q DEV_AUTH=%q: logged at %q, want nothing", mode, dev, level)
				}
			}
		}
	}
}

// The compose stack keeps sign-up: SIGNUP_MODE=open with DEV_AUTH="1" (docker-compose.yml).
func TestSignupModeComposeStaysOpen(t *testing.T) {
	t.Setenv("SIGNUP_MODE", "open")
	t.Setenv("DEV_AUTH", "1")
	if got := LoadConfig().Auth.Signup; got != SignupOpen {
		t.Fatalf("compose env resolved to %q, want open", got)
	}
}

func TestSeatCapDefault(t *testing.T) {
	for in, want := range map[string]int{"": DefaultSeatCap, "15": 15, "3": 3, "0": DefaultSeatCap, "-2": DefaultSeatCap, "x": DefaultSeatCap} {
		t.Setenv("SEAT_CAP", in)
		if got := LoadConfig().Auth.SeatCap; got != want {
			t.Errorf("SEAT_CAP=%q: %d, want %d", in, got, want)
		}
	}
}

func setOrUnset(t *testing.T, key, val, unset string) {
	t.Helper()
	if val == unset {
		t.Setenv(key, "") // registers the restore
		_ = os.Unsetenv(key)
		return
	}
	t.Setenv(key, val)
}

// logLevelOf runs LogSignupMode against a capturing handler: the level logged, or "".
func logLevelOf(a AuthConfig) string {
	var buf bytes.Buffer
	a.LogSignupMode(slog.New(slog.NewTextHandler(&buf, nil)))
	out := buf.String()
	switch {
	case out == "":
		return ""
	case strings.Contains(out, "level=ERROR"):
		return "ERROR"
	case strings.Contains(out, "level=INFO"):
		return "INFO"
	default:
		return out
	}
}
