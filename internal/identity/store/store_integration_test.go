package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
)

// Integration test against a real Postgres, gated on XLEARN_TEST_DATABASE_URL so
// CI (which has no database) skips it. Run locally, e.g.:
//
//	XLEARN_TEST_DATABASE_URL=postgres://xlearn_identity:pw@localhost:5433/xlearndb?sslmode=disable \
//	  go test ./internal/identity/store/ -run TestStore -count=1
//
// The DSN must be for the xlearn_identity role (search_path identity, owning only
// schema identity) so the test also exercises the least-privilege model.
func TestStoreIntegration(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the identity store integration test")
	}
	ctx := context.Background()

	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Idempotent re-run (advisory lock + goose versioning).
	if err := store.Migrate(ctx, dsn, testLogger()); err != nil {
		t.Fatalf("migrate (rerun): %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	st := store.New(pool)

	if err := st.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	in := store.OAuthUpsert{
		Provider:       "github",
		ProviderUserID: "gh-" + newTestID(),
		DisplayName:    "Ada Lovelace",
		Email:          "ada@example.com",
	}

	acct, created, err := st.FindOrCreateAccount(ctx, in)
	if err != nil {
		t.Fatalf("first FindOrCreateAccount: %v", err)
	}
	if !created {
		t.Fatalf("first sign-in should report created=true")
	}
	if acct.ID == "" || acct.DisplayName != "Ada Lovelace" || acct.Email != "ada@example.com" {
		t.Fatalf("unexpected account: %+v", acct)
	}

	// Second call with the same provider identity resolves to the same account.
	acct2, created2, err := st.FindOrCreateAccount(ctx, in)
	if err != nil {
		t.Fatalf("second FindOrCreateAccount: %v", err)
	}
	if created2 {
		t.Fatalf("returning sign-in should report created=false")
	}
	if acct2.ID != acct.ID {
		t.Fatalf("returning sign-in got a different account: %s != %s", acct2.ID, acct.ID)
	}

	// Onboarding row was created on first sign-in; path starts empty.
	ob, err := st.GetOnboarding(ctx, acct.ID)
	if err != nil {
		t.Fatalf("GetOnboarding: %v", err)
	}
	if ob.PathChosen != "" || ob.BudgetSet {
		t.Fatalf("fresh onboarding should be empty: %+v", ob)
	}

	ob, err = st.SetOnboardingPath(ctx, acct.ID, "dsa")
	if err != nil {
		t.Fatalf("SetOnboardingPath: %v", err)
	}
	if ob.PathChosen != "dsa" {
		t.Fatalf("path not persisted: %+v", ob)
	}

	// S10 — partial account update (PATCH /me). Only the provided fields change.
	name := "Sujay Kumar"
	tz := "Asia/Kolkata"
	budget := []byte(`{"weekday_minutes":120,"weekend_band":"5"}`)
	reminders := []byte(`{"daily_reminder_on":true,"daily_reminder_time":"20:00","revision_due_alerts_on":false}`)
	upd, err := st.UpdateAccount(ctx, acct.ID, store.AccountUpdate{
		DisplayName: &name, Timezone: &tz, StudyBudget: budget, Reminders: reminders,
	})
	if err != nil {
		t.Fatalf("UpdateAccount: %v", err)
	}
	if upd.DisplayName != name || upd.Timezone != tz {
		t.Fatalf("update not applied: %+v", upd)
	}
	if string(upd.StudyBudget) == "" || string(upd.Reminders) == "" {
		t.Fatalf("budget/reminders not stored: %+v", upd)
	}
	// A partial update (name only) must leave the timezone + budget untouched.
	// (jsonb reorders keys + adds whitespace, so compare parsed values, not raw bytes.)
	rename := "Renamed"
	upd2, err := st.UpdateAccount(ctx, acct.ID, store.AccountUpdate{DisplayName: &rename})
	if err != nil {
		t.Fatalf("partial UpdateAccount: %v", err)
	}
	if upd2.DisplayName != rename || upd2.Timezone != tz || weekdayMinutes(t, upd2.StudyBudget) != 120 {
		t.Fatalf("partial update changed untouched columns: %+v", upd2)
	}

	// S10 — onboarding budget step writes the account budget + flips budget_set (one tx).
	obBudget, err := st.SetOnboardingBudget(ctx, acct.ID, []byte(`{"weekday_minutes":60,"weekend_band":"2"}`))
	if err != nil {
		t.Fatalf("SetOnboardingBudget: %v", err)
	}
	if !obBudget.BudgetSet {
		t.Fatalf("budget_set not persisted: %+v", obBudget)
	}
	if acctAfter, _ := st.GetAccount(ctx, acct.ID); weekdayMinutes(t, acctAfter.StudyBudget) != 60 {
		t.Fatalf("onboarding budget not written to account: %s", acctAfter.StudyBudget)
	}

	// S10 — Finish stamps completed_at, idempotently.
	obDone, err := st.CompleteOnboarding(ctx, acct.ID)
	if err != nil {
		t.Fatalf("CompleteOnboarding: %v", err)
	}
	if obDone.CompletedAt.IsZero() {
		t.Fatalf("completion state wrong: %+v", obDone)
	}
	obDone2, err := st.CompleteOnboarding(ctx, acct.ID)
	if err != nil {
		t.Fatalf("re-CompleteOnboarding: %v", err)
	}
	if !obDone2.CompletedAt.Equal(obDone.CompletedAt) {
		t.Fatalf("completed_at moved on re-finish: %v != %v", obDone2.CompletedAt, obDone.CompletedAt)
	}

	// Session lifecycle.
	sid := "sess-" + newTestID()
	sess, err := st.CreateSession(ctx, sid, acct.ID, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	got, err := st.GetValidSession(ctx, sess.ID)
	if err != nil {
		t.Fatalf("GetValidSession: %v", err)
	}
	if got.AccountID != acct.ID {
		t.Fatalf("session account mismatch: %s != %s", got.AccountID, acct.ID)
	}
	revoked, err := st.RevokeSession(ctx, sess.ID)
	if err != nil || !revoked {
		t.Fatalf("RevokeSession: revoked=%v err=%v", revoked, err)
	}
	if _, err := st.GetValidSession(ctx, sess.ID); err == nil {
		t.Fatalf("revoked session should not validate")
	}

	// Outbox: the account_created row from first sign-in is unsent and relayable.
	rows, err := st.ListUnsentOutbox(ctx, 100)
	if err != nil {
		t.Fatalf("ListUnsentOutbox: %v", err)
	}
	var mine *store.OutboxRow
	for i := range rows {
		if containsAccount(rows[i].Payload, acct.ID) {
			mine = &rows[i]
			break
		}
	}
	if mine == nil {
		t.Fatalf("no account_created outbox row for %s", acct.ID)
	}
	if mine.Subject != store.SubjectAccountCreated {
		t.Fatalf("unexpected subject: %s", mine.Subject)
	}
	if err := st.MarkOutboxSent(ctx, mine.EventID); err != nil {
		t.Fatalf("MarkOutboxSent: %v", err)
	}
	rows2, err := st.ListUnsentOutbox(ctx, 100)
	if err != nil {
		t.Fatalf("ListUnsentOutbox after mark: %v", err)
	}
	for _, r := range rows2 {
		if r.EventID == mine.EventID {
			t.Fatalf("outbox row %s still unsent after MarkOutboxSent", mine.EventID)
		}
	}
}
