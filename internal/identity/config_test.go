package identity

import (
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
