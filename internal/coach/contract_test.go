package coach

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
)

// This file is m1-08's PRECONDITION, enforced in CI from m1-10 on.
//
// m1-08 (M1c) drops api_key_config.is_default. A contract migration is only safe once
// nothing reads or writes the column: a surviving reader would start erroring the moment
// the drop lands, in production, on the coach path. Reviewing for that by eye does not
// scale across releases, so it is a test.
//
// It pairs with hack/lint-dropped-columns.sh, which scans the .sql and the sqlc-generated
// SQL. This one scans the hand-written GO, which that script does not read. Between them:
// no query, no generated SQL and no Go identifier touches is_default.

// isDefaultPattern matches the column and its Go spelling as whole words.
var isDefaultPattern = regexp.MustCompile(`(?i)(^|[^a-z0-9_])is_?default([^a-z0-9_]|$)`)

// starSelectPattern matches a star select or returning against coach.api_key_config.
// sqlc expands a star to the column set it saw at generate time, so a star query on this
// table reads is_default positionally WITHOUT naming it — which both hides it from a
// name-based grep and breaks the instant M1c drops the column.
var starSelectPattern = regexp.MustCompile(`(?is)(SELECT|RETURNING)\s+\*\s*(FROM\s+coach\.api_key_config|;)`)

// TestNoIsDefaultColumnInProductionCode fails if the COLUMN NAME `is_default` appears in
// any of coach's own (non-test) Go or SQL source, outside the places it legitimately
// survives:
//
//   - store/migrations/   — history, never edited; 00003 created the column and 00004
//     made it nullable;
//   - store/schema.sql    — the sqlc schema declaration;
//   - store/gen/          — generated from the queries. Nothing in the queries names the
//     column any more, so the only mention left is the struct field in models.go, which
//     exists because the column still does; m1-08's regenerate removes it.
//
// Scope, and why it is drawn here:
//
//   - The GO IDENTIFIER `IsDefault` is allowed. It is a DIFFERENT THING with a
//     confusingly similar name: KeyConfig.IsDefault and keyViewJSON.IsDefault are the
//     flag derived from coach.key_default(feature='coach'), and `json:"is_default"` is
//     the v1 API field name both must keep so a v1.6.0 tab left open across the upgrade
//     still works. Banning the identifier would ban the v1 contract.
//   - TEST files are exempt, because the tests that prove the column is dead have to name
//     it: internal/coach/store/m1a_integration_test.go writes a stale is_default
//     directly and asserts the readers ignore it, and counts the column to prove no
//     writer sets it.
//
// This grep is one of three overlapping checks, each covering what the others cannot:
//
//	this test                          coach's hand-written Go + SQL
//	hack/lint-dropped-columns.sh       the .sql queries AND the sqlc-GENERATED SQL, where
//	                                   a `SELECT *` would read the column without naming it
//	m1a_integration_test.go            the actual behaviour against Postgres: a stale
//	                                   is_default is ignored, and no writer sets it
//
// Together they are m1-08's "no reader or writer left" precondition for coach.
func TestNoIsDefaultColumnInProductionCode(t *testing.T) {
	root := "." // internal/coach

	exempt := func(rel string) bool {
		rel = filepath.ToSlash(rel)
		switch {
		case strings.HasPrefix(rel, "store/migrations/"):
			return true
		case rel == "store/schema.sql":
			return true
		case strings.HasPrefix(rel, "store/gen/"):
			return true
		case strings.HasSuffix(rel, "_test.go"):
			return true
		}
		return false
	}

	var offenders []string
	scanned := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".go" && ext != ".sql" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if exempt(rel) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		for i, line := range strings.Split(string(b), "\n") {
			// Comments may discuss the column — retiring it is this sprint's whole
			// story — so only code counts.
			code := line
			if idx := strings.Index(code, "//"); idx >= 0 {
				code = code[:idx]
			}
			if ext == ".sql" {
				if idx := strings.Index(code, "--"); idx >= 0 {
					code = code[:idx]
				}
			}
			if !strings.Contains(code, "is_default") {
				continue
			}
			// The v1 JSON field name is not the column.
			if strings.Contains(code, `json:"is_default"`) {
				continue
			}
			offenders = append(offenders,
				filepath.ToSlash(rel)+":"+itoaTest(i+1)+": "+strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if scanned == 0 {
		t.Fatal("no files scanned; the test is not looking where it thinks")
	}
	if len(offenders) > 0 {
		t.Fatalf("the is_default COLUMN is referenced in coach's production code — m1-08 cannot drop it:\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// TestNoQueryNamesIsDefault is the narrow, high-value half: not one of coach's queries may
// name the column, in any direction. (The generated SQL is covered by
// hack/lint-dropped-columns.sh, which also catches a `SELECT *` expanding to it.)
func TestNoQueryNamesIsDefault(t *testing.T) {
	dir := filepath.Join("store", "queries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read queries dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if idx := strings.Index(line, "--"); idx >= 0 {
				line = line[:idx]
			}
			if isDefaultPattern.MatchString(line) {
				t.Errorf("%s:%d names is_default: %q", e.Name(), i+1, strings.TrimSpace(line))
			}
		}
	}
}

// TestNoStarSelectOnApiKeyConfig: every query against coach.api_key_config must list its
// columns. A `SELECT *` generates positional scanning code against today's column set, so
// the M1c drop turns it into a runtime error on the coach path — and the star hides the
// dropped column from any name-based grep.
func TestNoStarSelectOnApiKeyConfig(t *testing.T) {
	dir := filepath.Join("store", "queries")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read queries dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		found = true
		// Strip -- comments so prose about `SELECT *` doesn't trip the scan.
		var sb strings.Builder
		for _, line := range strings.Split(string(b), "\n") {
			if idx := strings.Index(line, "--"); idx >= 0 {
				line = line[:idx]
			}
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
		sql := sb.String()
		if !strings.Contains(sql, "api_key_config") {
			continue
		}
		if m := starSelectPattern.FindString(sql); m != "" {
			t.Errorf("%s has a star select/returning touching coach.api_key_config: %q", e.Name(), strings.Join(strings.Fields(m), " "))
		}
		// A bare `SELECT *` anywhere in a file that touches the table is enough to worry
		// about: catch it even when the FROM is on another line.
		for _, stmt := range strings.Split(sql, ";") {
			if !strings.Contains(stmt, "api_key_config") {
				continue
			}
			if regexp.MustCompile(`(?is)(SELECT|RETURNING)\s+\*`).MatchString(stmt) {
				t.Errorf("%s: a statement touching coach.api_key_config selects or returns *", e.Name())
			}
		}
	}
	if !found {
		t.Fatal("no query files scanned; the test is not looking where it thinks")
	}
}

// TestProviderRegistryMirrorsTheDatabaseCheck is the three-way agreement the plan asks
// for: the Service's provider REGISTRY (what the handlers validate against and GET /models
// enumerates), store.ValidProvider (a static mirror, because store cannot import coach
// without a cycle), and the `CHECK (provider IN (...))` in migration 00001.
//
// Without this test the three can drift silently in the worst direction: a provider added
// to the registry but not the CHECK lets a learner paste a key the INSERT then rejects,
// and one added to the CHECK but not the registry stores a key no chat can use.
func TestProviderRegistryMirrorsTheDatabaseCheck(t *testing.T) {
	svc := newProviderOnlyService(t)

	registry := map[string]bool{}
	for _, id := range svc.ProviderIDs() {
		registry[id] = true
	}

	// The CHECK, read from the migration rather than restated here — restating it would
	// make this test agree with itself instead of with the schema.
	check := providersFromMigrationCheck(t)

	if len(registry) == 0 || len(check) == 0 {
		t.Fatalf("nothing to compare (registry=%v check=%v)", registry, check)
	}
	for id := range registry {
		if !check[id] {
			t.Errorf("provider %q is in the registry but not in the DB CHECK: a stored key would be rejected by the INSERT", id)
		}
		if !store.ValidProvider(id) {
			t.Errorf("provider %q is in the registry but not in store.ValidProvider", id)
		}
	}
	for id := range check {
		if !registry[id] {
			t.Errorf("provider %q is in the DB CHECK but has no client in the registry: a key would be unusable", id)
		}
		if !store.ValidProvider(id) {
			t.Errorf("provider %q is in the DB CHECK but not in store.ValidProvider", id)
		}
	}
	// And the mirror lists nothing extra.
	for _, id := range []string{store.ProviderOpenAI, store.ProviderAnthropic} {
		if !registry[id] || !check[id] {
			t.Errorf("store.ValidProvider accepts %q, which is missing from the registry or the CHECK", id)
		}
	}
	if store.ValidProvider("google") || store.ValidProvider("") {
		t.Error("store.ValidProvider accepts a provider the CHECK does not")
	}
}

// providersFromMigrationCheck extracts the provider ids from the CHECK constraint in the
// coach migrations, so the test reads the schema rather than a copy of it.
func providersFromMigrationCheck(t *testing.T) map[string]bool {
	t.Helper()
	dir := filepath.Join("store", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	re := regexp.MustCompile(`(?is)CHECK\s*\(\s*provider\s+IN\s*\(([^)]*)\)`)
	out := map[string]bool{}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		m := re.FindStringSubmatch(string(b))
		if m == nil {
			continue
		}
		for _, raw := range strings.Split(m[1], ",") {
			if id := strings.Trim(strings.TrimSpace(raw), "'\""); id != "" {
				out[id] = true
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("no provider CHECK found in the coach migrations")
	}
	return out
}

// newProviderOnlyService builds a Service with both provider clients and nothing else
// wired — enough for the registry and catalog assertions.
func newProviderOnlyService(t *testing.T) *Service {
	t.Helper()
	openai := NewOpenAIProvider("http://127.0.0.1:1", nil)
	anthropic := NewAnthropicProvider("http://127.0.0.1:1", nil)
	return NewService(newMemStore(), fakeVerifier{subject: "acct"}, testCipher(), nil,
		openai, anthropic, coursetest.Registry(t), discardLogger())
}

func itoaTest(n int) string { return itoa(n) }
