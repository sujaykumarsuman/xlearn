package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/course/coursetest"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store/gen"
)

// m1-02 (M1a expand): every v1.6.0 create path writes admitted_via = 'dev', the new
// account columns take their constant defaults, enrollments take the manifest's public
// default on insert only, and admin_audit exists. Nothing reads these in v1.6.0.
func TestM1aIdentityColumns(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the identity store integration test")
	}
	ctx := context.Background()
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	type cols struct {
		role, status, visibility string
		admittedVia              *string
	}
	read := func(id string) cols {
		t.Helper()
		var c cols
		if err := pool.QueryRow(ctx,
			`SELECT role, status, admitted_via, profile_visibility FROM identity.account WHERE id = $1`, id,
		).Scan(&c.role, &c.status, &c.admittedVia, &c.visibility); err != nil {
			t.Fatalf("read account: %v", err)
		}
		return c
	}
	check := func(what string, c cols) {
		t.Helper()
		if c.role != "learner" || c.status != "active" || c.visibility != "public" ||
			c.admittedVia == nil || *c.admittedVia != store.AdmittedViaDev {
			t.Fatalf("%s: %+v (admitted_via %v), want learner/active/public/dev", what, c, c.admittedVia)
		}
	}

	oauth, created, err := st.FindOrCreateAccount(ctx, store.OAuthUpsert{
		Provider: "github", ProviderUserID: "m1a-" + newTestID(), DisplayName: "M1a OAuth",
	})
	if err != nil || !created {
		t.Fatalf("oauth create: created=%v err=%v", created, err)
	}
	check("oauth/dev create", read(oauth.ID))

	email, err := st.CreateEmailAccount(ctx, "m1a-"+newTestID()+"@example.com", "$2a$10$abcdefghijklmnopqrstuv", "M1a Email")
	if err != nil {
		t.Fatalf("email create: %v", err)
	}
	check("email signup", read(email.ID))

	// Enrollment: the default the caller passes (the manifest's: dsa public, behavioral
	// private) is written on insert …
	visible := func(slug string) bool {
		t.Helper()
		var v bool
		if err := pool.QueryRow(ctx,
			`SELECT public_visible FROM identity.path_enrollment WHERE account_id = $1 AND path_slug = $2`, email.ID, slug,
		).Scan(&v); err != nil {
			t.Fatalf("read enrollment %s: %v", slug, err)
		}
		return v
	}
	for slug, want := range map[string]bool{"dsa": true, "behavioral": false} {
		if _, err := st.StartEnrollment(ctx, email.ID, slug, coursetest.All(t)[slug].PublicStats.Visible()); err != nil {
			t.Fatalf("enroll %s: %v", slug, err)
		}
		if got := visible(slug); got != want {
			t.Fatalf("%s public_visible = %v, want %v", slug, got, want)
		}
	}
	// … and never on a re-start: the learner's own choice survives.
	if _, err := pool.Exec(ctx, `UPDATE identity.path_enrollment SET public_visible = false WHERE account_id = $1 AND path_slug = 'dsa'`, email.ID); err != nil {
		t.Fatalf("toggle: %v", err)
	}
	if _, err := st.StartEnrollment(ctx, email.ID, "dsa", true); err != nil {
		t.Fatalf("re-enroll: %v", err)
	}
	if visible("dsa") {
		t.Fatal("a re-start overwrote public_visible")
	}

	// The CHECKs hold: an unknown role / status / admission is rejected.
	for _, q := range []string{
		`UPDATE identity.account SET role = 'admin' WHERE id = $1`,
		`UPDATE identity.account SET status = 'banned' WHERE id = $1`,
		`UPDATE identity.account SET admitted_via = 'waitlist' WHERE id = $1`,
		`UPDATE identity.account SET profile_visibility = 'friends' WHERE id = $1`,
	} {
		if _, err := pool.Exec(ctx, q, email.ID); err == nil {
			t.Fatalf("CHECK accepted: %s", q)
		}
	}

	// admin_audit: insert + newest-first list (m1-04's CLI writes it).
	q := gen.New(pool)
	row, err := q.InsertAdminAudit(ctx, gen.InsertAdminAuditParams{Verb: "m1a-test", Target: email.ID, Detail: []byte(`{"k":"v"}`)})
	if err != nil {
		t.Fatalf("insert admin_audit: %v", err)
	}
	rows, err := q.ListAdminAudit(ctx, 5)
	if err != nil || len(rows) == 0 || rows[0].ID != row.ID {
		t.Fatalf("list admin_audit: %v rows=%d", err, len(rows))
	}
}
