package admin

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/sujaykumarsuman/xlearn/internal/identity"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// The admin CLI against PG 18 (CI's store-integration job; skipped without
// XLEARN_TEST_DATABASE_URL): every verb's effect, its admin_audit row (same transaction;
// refusals audited too), the last-owner and seat guards, and no secret or email in the
// audit rows or the stderr log. The shared test database may hold other tests' accounts,
// so the test normalizes owners first and sizes SEAT_CAP from the live seat count.
func TestAdminCLIIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the identity admin CLI integration test")
	}
	ctx := context.Background()
	// The TEST migrates; the CLI never does.
	if err := store.Migrate(ctx, dsn, slog.New(slog.NewJSONHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	pg := store.New(pool)
	adm := store.NewAdmin(pool)

	// Test-only: nobody else is an active owner, so the last-owner guard is testable.
	if _, err := pool.Exec(ctx, `UPDATE identity.account SET role = 'learner' WHERE role = 'owner'`); err != nil {
		t.Fatal(err)
	}

	id := hexID()
	mkAcct := func(name string) store.Account {
		t.Helper()
		a, err := pg.CreateEmailAccount(ctx, name+"-"+id+"@example.com", "$2a$10$abcdefghijklmnopqrstuv", name)
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
		if _, err := pg.SetUsername(ctx, a.ID, name+id); err != nil {
			t.Fatalf("username %s: %v", name, err)
		}
		return a
	}
	a, b, c := mkAcct("ada"), mkAcct("bob"), mkAcct("cy")
	for _, sid := range []string{"a1", "a2", "b1"} {
		owner := a.ID
		if sid == "b1" {
			owner = b.ID
		}
		if _, err := pg.CreateSession(ctx, sid+"-"+id, owner, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}

	seatCap := 15
	var stderrAll bytes.Buffer
	run := func(args ...string) (int, string) {
		t.Helper()
		var out, errb bytes.Buffer
		code := Run(ctx, Deps{
			Open:         func(context.Context) (Store, func(), error) { return adm, nil, nil },
			SeatCap:      seatCap,
			HashPassword: identity.HashPassword,
			NewPassword:  NewPassword,
		}, args, &out, &errb)
		stderrAll.Write(errb.Bytes())
		// Every run logs exactly one JSON line on stderr (plus a human error line on failure).
		if !strings.Contains(errb.String(), `"msg":"identity admin"`) {
			t.Fatalf("%q: no log line on stderr: %s", args, errb.String())
		}
		return code, out.String()
	}
	lastAudit := func() store.AuditRow {
		t.Helper()
		rows, err := adm.ListAudit(ctx, 1)
		if err != nil || len(rows) != 1 {
			t.Fatalf("audit: %v %v", rows, err)
		}
		return rows[0]
	}
	expectAudit := func(verb, target, outcome string) store.AuditRow {
		t.Helper()
		r := lastAudit()
		if r.Verb != verb || r.Target != target || r.Detail["outcome"] != outcome {
			t.Fatalf("audit = %s %s %v, want %s %s outcome=%s", r.Verb, r.Target, r.Detail, verb, target, outcome)
		}
		return r
	}
	sessionValid := func(sid string) bool {
		_, err := pg.GetValidSession(ctx, sid+"-"+id)
		return err == nil
	}
	roleStatus := func(acct string) (string, string) {
		t.Helper()
		var role, status string
		if err := pool.QueryRow(ctx, `SELECT role, status FROM identity.account WHERE id = $1`, acct).Scan(&role, &status); err != nil {
			t.Fatal(err)
		}
		return role, status
	}

	// list: JSON, filtered, audited (a read, target '-').
	code, out := run("account", "list", "--json", "--role", "learner", "--status", "active")
	var listed []store.AdminAccount
	if code != ExitOK || json.Unmarshal([]byte(out), &listed) != nil {
		t.Fatalf("list --json: %d %s", code, out)
	}
	found := false
	for _, l := range listed {
		found = found || l.ID == a.ID
		if l.Role != "learner" || l.Status != "active" {
			t.Fatalf("filter leaked %+v", l)
		}
	}
	if !found {
		t.Fatal("list lacks ada")
	}
	if r := expectAudit(store.VerbAccountList, "-", "ok"); r.Detail["role"] != "learner" || r.Detail["count"] == nil {
		t.Fatalf("list audit detail %v", r.Detail)
	}
	if code, out := run("account", "list", "--role", "owner"); code != ExitOK || !strings.Contains(out, "0 account(s)") {
		t.Fatalf("list owners: %d %s", code, out)
	}
	// --dormant: cy never signed in, ada did an instant ago.
	code, out = run("account", "list", "--dormant", "30d", "--json")
	if code != ExitOK || !strings.Contains(out, c.ID) || strings.Contains(out, a.ID) {
		t.Fatalf("list --dormant 30d: %d, want cy and not ada: %s", code, out)
	}

	// set-role: promote by email (case-insensitive): the first owner.
	if code, out := run("account", "set-role", strings.ToUpper("ada-"+id+"@example.com"), "owner"); code != ExitOK || !strings.Contains(out, "learner -> owner") {
		t.Fatalf("set-role owner: %d %s", code, out)
	}
	if r := expectAudit(store.VerbAccountSetRole, a.ID, "ok"); r.Detail["from_role"] != "learner" || r.Detail["to_role"] != "owner" {
		t.Fatalf("set-role audit %v", r.Detail)
	}
	// admitted_via never changes.
	var via string
	_ = pool.QueryRow(ctx, `SELECT admitted_via FROM identity.account WHERE id = $1`, a.ID).Scan(&via)
	if via != store.AdmittedViaDev {
		t.Fatalf("set-role changed admitted_via to %q", via)
	}

	// The last active owner: suspend and demote both refused (exit 4), audited, no change.
	if code, _ := run("account", "suspend", a.ID); code != ExitRefused {
		t.Fatalf("suspend last owner: exit %d, want %d", code, ExitRefused)
	}
	expectAudit(store.VerbAccountSuspend, a.ID, "refused_last_owner")
	if code, _ := run("account", "set-role", a.ID, "tester"); code != ExitRefused {
		t.Fatalf("demote last owner: exit %d", code)
	}
	expectAudit(store.VerbAccountSetRole, a.ID, "refused_last_owner")
	if role, status := roleStatus(a.ID); role != "owner" || status != "active" || !sessionValid("a1") {
		t.Fatalf("refused verbs changed ada: %s/%s", role, status)
	}

	// A second owner (by username, case-insensitive), then suspend kills ada's sessions at once.
	if code, _ := run("account", "set-role", strings.ToUpper("bob"+id), "owner"); code != ExitOK {
		t.Fatalf("set-role bob owner: %d", code)
	}
	code, out = run("account", "suspend", "ada-"+id+"@example.com")
	if code != ExitOK || !strings.Contains(out, "revoked 2 session(s)") {
		t.Fatalf("suspend: %d %s", code, out)
	}
	if r := expectAudit(store.VerbAccountSuspend, a.ID, "ok"); r.Detail["revoked_sessions"] != float64(2) {
		t.Fatalf("suspend audit %v", r.Detail)
	}
	if sessionValid("a1") || sessionValid("a2") {
		t.Fatal("a session survived the suspend")
	}
	if _, st := roleStatus(a.ID); st != "suspended" {
		t.Fatalf("status %s", st)
	}
	// bob is now the last ACTIVE owner (ada is suspended).
	if code, _ := run("account", "set-role", b.ID, "learner"); code != ExitRefused {
		t.Fatalf("demote the last active owner: exit %d", code)
	}
	// reactivate the owner (outside the seat cap: no seat check even at cap 1).
	seatCap = 1
	if code, out := run("account", "reactivate", a.ID); code != ExitOK || !strings.Contains(out, "reactivated") {
		t.Fatalf("reactivate owner: %d %s", code, out)
	}
	seatCap = 15

	// revoke-sessions.
	if code, out := run("account", "revoke-sessions", b.ID); code != ExitOK || !strings.Contains(out, "revoked 1 session(s)") {
		t.Fatalf("revoke-sessions: %d %s", code, out)
	}
	expectAudit(store.VerbAccountRevokeSessions, b.ID, "ok")
	if sessionValid("b1") {
		t.Fatal("bob's session survived revoke-sessions")
	}

	// Seats: the cap binds reactivate of a learner and set-role … learner.
	code, out = run("seats")
	if code != ExitOK || !strings.Contains(out, "invites: n/a until L-A") {
		t.Fatalf("seats: %d %s", code, out)
	}
	used := expectAudit(store.VerbSeats, "-", "ok").Detail["used"].(float64)
	if code, _ := run("account", "suspend", c.ID); code != ExitOK { // cy: a learner
		t.Fatalf("suspend cy: %d", code)
	}
	seatCap = int(used) - 1 // every seat but cy's is taken → full
	if code, _ := run("account", "reactivate", c.ID); code != ExitRefused {
		t.Fatalf("reactivate over the cap: exit %d", code)
	}
	if r := expectAudit(store.VerbAccountReactivate, c.ID, "refused_seat_cap"); r.Detail["seat_cap"] != float64(seatCap) {
		t.Fatalf("seat audit %v", r.Detail)
	}
	if code, _ := run("account", "set-role", b.ID, "learner"); code != ExitRefused { // bob active owner → learner takes a seat
		t.Fatalf("set-role learner over the cap: exit %d", code)
	}
	expectAudit(store.VerbAccountSetRole, b.ID, "refused_seat_cap")
	seatCap = int(used)
	if code, _ := run("account", "reactivate", c.ID); code != ExitOK {
		t.Fatalf("reactivate within the cap: %d", code)
	}
	seatCap = int(used) + 10
	if code, _ := run("account", "set-role", b.ID, "learner"); code != ExitOK {
		t.Fatalf("set-role learner within the cap: %d", code)
	}

	// create --role tester: the password is printed once, never logged or audited.
	email := "tess-" + id + "@example.com"
	code, out = run("account", "create", "--role", "tester", "--email", strings.ToUpper(email))
	if code != ExitOK {
		t.Fatalf("create: %d %s", code, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	pw := lines[len(lines)-1]
	if len(pw) != 24 || strings.Count(out, pw) != 1 {
		t.Fatalf("create output %q: want the 24-char password exactly once", out)
	}
	created := lastAudit()
	if created.Verb != store.VerbAccountCreate || created.Detail["outcome"] != "ok" {
		t.Fatalf("create audit %+v", created)
	}
	var tester struct {
		id, role, via string
		hash          string
		onboarding    int
		outbox        int
	}
	if err := pool.QueryRow(ctx, `SELECT id::text, role, admitted_via, password_hash FROM identity.account WHERE lower(email) = $1`, email).
		Scan(&tester.id, &tester.role, &tester.via, &tester.hash); err != nil {
		t.Fatalf("tester row: %v", err)
	}
	if created.Target != tester.id || tester.role != "tester" || tester.via != "cli" {
		t.Fatalf("tester %+v, audit target %s", tester, created.Target)
	}
	if bcrypt.CompareHashAndPassword([]byte(tester.hash), []byte(pw)) != nil {
		t.Fatal("the printed password doesn't match the stored hash")
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM identity.onboarding WHERE account_id = $1`, tester.id).Scan(&tester.onboarding)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM identity.outbox WHERE subject = $1 AND payload_json::text LIKE '%' || $2 || '%'`,
		store.SubjectAccountCreated, tester.id).Scan(&tester.outbox)
	if tester.onboarding != 1 || tester.outbox != 1 {
		t.Fatalf("tester onboarding rows %d, outbox rows %d; want 1, 1", tester.onboarding, tester.outbox)
	}
	if code, _ := run("account", "create", "--role", "tester", "--email", email); code != ExitRefused {
		t.Fatalf("duplicate create: exit %d", code)
	}
	expectAudit(store.VerbAccountCreate, "-", "refused_email_taken")

	// Unknown user: exit 3, audited without echoing the argument.
	if code, _ := run("account", "suspend", "ghost-"+id+"@example.com"); code != ExitNotFound {
		t.Fatalf("unknown user: exit %d", code)
	}
	expectAudit(store.VerbAccountSuspend, "-", "user_not_found")

	// No secret and no email anywhere in this run's audit rows or stderr lines.
	rows, err := adm.ListAudit(ctx, 60)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		raw, _ := json.Marshal(r)
		if strings.Contains(string(raw), pw) || strings.Contains(string(raw), "@") || strings.Contains(string(raw), id) {
			t.Fatalf("audit row leaks a secret, an email or a user argument: %s", raw)
		}
	}
	if strings.Contains(stderrAll.String(), pw) {
		t.Fatal("the tester password reached stderr")
	}

	// Cleanup for later runs: no owner left behind.
	_, _ = pool.Exec(ctx, `UPDATE identity.account SET role = 'learner' WHERE role = 'owner'`)
}

func hexID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
