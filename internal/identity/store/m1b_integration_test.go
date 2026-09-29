package store_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// m1-04 (M1b): GetValidSession joins status = 'active' and returns the account's role,
// status and accepted_at with the SESSION's created_at; RevokeAllSessions and a password
// change end every live session of the account and no other.
func TestSessionStatusJoinAndRevokeAll(t *testing.T) {
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

	a, err := st.CreateEmailAccount(ctx, "m1b-"+newTestID()+"@example.com", "$2a$10$abcdefghijklmnopqrstuv", "M1b")
	if err != nil {
		t.Fatal(err)
	}
	other, err := st.CreateEmailAccount(ctx, "m1b-"+newTestID()+"@example.com", "$2a$10$abcdefghijklmnopqrstuv", "Other")
	if err != nil {
		t.Fatal(err)
	}
	if a.Role != store.RoleLearner || a.Status != store.StatusActive || a.AdmittedVia != store.AdmittedViaDev {
		t.Fatalf("new account %+v, want learner/active/dev", a)
	}
	sid := func(s string) string { return s + "-" + a.ID }
	for _, s := range []string{"one", "two", "three"} {
		if _, err := st.CreateSession(ctx, sid(s), a.ID, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	otherSID := "keep-" + other.ID
	if _, err := st.CreateSession(ctx, otherSID, other.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	sess, err := st.GetValidSession(ctx, sid("one"))
	if err != nil {
		t.Fatalf("GetValidSession: %v", err)
	}
	if sess.AccountID != a.ID || sess.Role != "learner" || sess.Status != "active" || !sess.AcceptedAt.IsZero() ||
		sess.CreatedAt.IsZero() || time.Since(sess.CreatedAt) > time.Minute {
		t.Fatalf("session %+v", sess)
	}
	// The role is read live from the account row.
	if _, err := pool.Exec(ctx, `UPDATE identity.account SET role = 'tester' WHERE id = $1`, a.ID); err != nil {
		t.Fatal(err)
	}
	if sess, _ := st.GetValidSession(ctx, sid("one")); sess.Role != "tester" {
		t.Fatalf("role after set: %q", sess.Role)
	}

	// Suspended: every session fails validation at once, with no revocation step …
	if _, err := pool.Exec(ctx, `UPDATE identity.account SET status = 'suspended' WHERE id = $1`, a.ID); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"one", "two", "three"} {
		if _, err := st.GetValidSession(ctx, sid(s)); !errors.Is(err, store.ErrNotFound) {
			t.Fatalf("suspended session %s: %v, want ErrNotFound", s, err)
		}
	}
	// … and they come back on reactivation (suspend in the CLI also revokes them).
	if _, err := pool.Exec(ctx, `UPDATE identity.account SET status = 'active' WHERE id = $1`, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetValidSession(ctx, sid("two")); err != nil {
		t.Fatalf("after reactivation: %v", err)
	}

	n, err := st.RevokeAllSessions(ctx, a.ID)
	if err != nil || n != 3 {
		t.Fatalf("RevokeAllSessions = %d, %v; want 3", n, err)
	}
	if n, _ := st.RevokeAllSessions(ctx, a.ID); n != 0 {
		t.Fatalf("second RevokeAllSessions = %d, want 0", n)
	}
	if _, err := st.GetValidSession(ctx, otherSID); err != nil {
		t.Fatal("another account's session was revoked")
	}

	// A password change revokes every live session in the same transaction.
	if _, err := st.CreateSession(ctx, sid("four"), a.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	acct, revoked, err := st.SetAccountPassword(ctx, a.ID, "$2a$10$zyxwvutsrqponmlkjihgfe")
	if err != nil || revoked != 1 || acct.PasswordHash != "$2a$10$zyxwvutsrqponmlkjihgfe" {
		t.Fatalf("SetAccountPassword = %+v, %d, %v", acct, revoked, err)
	}
	if _, err := st.GetValidSession(ctx, sid("four")); err == nil {
		t.Fatal("a session survived the password change")
	}
	if _, err := st.GetValidSession(ctx, otherSID); err != nil {
		t.Fatal("the password change revoked another account's session")
	}
}

// m1-05 (P2): ListPublicVisibleCourses returns the account's enrollments with
// public_visible set, oldest first. Hidden enrollments and other accounts' are not listed,
// a re-start never flips the flag, and an account with none gets an empty slice.
func TestListPublicVisibleCourses(t *testing.T) {
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

	a, err := st.CreateEmailAccount(ctx, "m105-"+newTestID()+"@example.com", "$2a$10$abcdefghijklmnopqrstuv", "Visible")
	if err != nil {
		t.Fatal(err)
	}
	b, err := st.CreateEmailAccount(ctx, "m105-"+newTestID()+"@example.com", "$2a$10$abcdefghijklmnopqrstuv", "Other")
	if err != nil {
		t.Fatal(err)
	}
	if got, err := st.ListPublicVisibleCourses(ctx, a.ID); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("no enrollments: %#v, %v; want an empty slice", got, err)
	}
	for _, e := range []struct {
		acct, slug string
		visible    bool
	}{
		{a.ID, "course-a", true},
		{a.ID, "course-hidden", false},
		{a.ID, "course-b", true},
		{b.ID, "course-other", true},
	} {
		if _, err := st.StartEnrollment(ctx, e.acct, e.slug, e.visible); err != nil {
			t.Fatal(err)
		}
	}
	// A re-start keeps the learner's (here: the inserted) visibility.
	if _, err := st.StartEnrollment(ctx, a.ID, "course-hidden", true); err != nil {
		t.Fatal(err)
	}
	got, err := st.ListPublicVisibleCourses(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "course-a" || got[1] != "course-b" {
		t.Fatalf("ListPublicVisibleCourses = %v, want [course-a course-b]", got)
	}
	if _, err := st.ListPublicVisibleCourses(ctx, "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a malformed id: %v, want ErrNotFound", err)
	}
}
