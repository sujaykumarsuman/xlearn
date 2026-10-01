package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

// m1-07 (M1b, D27): the coach-assist record on an open attempt, the open-attempt read
// the gateway's coach mode gate uses, and the Assisted clamp in LogOutcome with its
// problem_solved `assist` field. Gated on XLEARN_TEST_DATABASE_URL like the other store
// integration tests.
func TestM1bCoachAssist(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the practice store integration test")
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

	// 00003 builds its index CONCURRENTLY: a failed build would leave it INVALID.
	var valid bool
	if err := pool.QueryRow(ctx, `SELECT i.indisvalid FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = 'practice' AND c.relname = 'attempt_open_by_account_idx'`).Scan(&valid); err != nil || !valid {
		t.Fatalf("attempt_open_by_account_idx valid=%v err=%v", valid, err)
	}

	// This test writes ~60 outbox rows; mark them sent afterwards so the other store tests
	// (TestStoreIntegration reads the first 100 unsent rows) never see a crowded outbox.
	// (A defer, not t.Cleanup: it must run before the deferred pool.Close.)
	var accounts []string
	defer func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE practice.outbox SET sent_at = now() WHERE sent_at IS NULL AND account_id = ANY($1::uuid[])`, accounts)
	}()

	openAttempt := func(t *testing.T, acct, problem string) store.OpenAttempt {
		t.Helper()
		accounts = append(accounts, acct)
		if _, err := st.StartAttempt(ctx, acct, problem, "dsa"); err != nil {
			t.Fatalf("start: %v", err)
		}
		atts, err := st.ListOpenAttempts(ctx, acct, problem)
		if err != nil || len(atts) != 1 {
			t.Fatalf("open attempts for %s: %v %v", problem, atts, err)
		}
		return atts[0]
	}

	t.Run("assist is idempotent, scoped to the account, and refused once concluded", func(t *testing.T) {
		acct := newTestUUID()
		att := openAttempt(t, acct, "16")
		if att.Purpose != store.PurposeCourse || att.ProblemID != "16" || att.PathSlug != "dsa" ||
			att.StageReached != store.StageAttempt || !att.CoachAssistAt.IsZero() || att.StartedAt.IsZero() {
			t.Fatalf("open attempt = %+v", att)
		}

		first, err := st.MarkCoachAssist(ctx, acct, att.ID)
		if err != nil || first.IsZero() {
			t.Fatalf("assist: %v %v", first, err)
		}
		again, err := st.MarkCoachAssist(ctx, acct, att.ID)
		if err != nil || !again.Equal(first) {
			t.Fatalf("second assist = %v %v, want the first time %v kept", again, err, first)
		}
		atts, _ := st.ListOpenAttempts(ctx, acct, "16")
		if len(atts) != 1 || !atts[0].CoachAssistAt.Equal(first) {
			t.Fatalf("open attempt after assist = %+v", atts)
		}
		s, err := st.GetState(ctx, acct, "16")
		if err != nil || !s.CoachAssistAt.Equal(first) {
			t.Fatalf("state.CoachAssistAt = %v %v, want %v", s.CoachAssistAt, err, first)
		}

		// Another account, an unknown id and a malformed id are all not found.
		for _, c := range []struct{ acct, id string }{
			{newTestUUID(), att.ID}, {acct, newTestUUID()}, {acct, "not-a-uuid"},
		} {
			if _, err := st.MarkCoachAssist(ctx, c.acct, c.id); !errors.Is(err, store.ErrNotFound) {
				t.Errorf("assist(%s, %s) = %v, want ErrNotFound", c.acct, c.id, err)
			}
		}

		res, err := st.LogOutcome(ctx, acct, "16", store.OutcomeMiss)
		if err != nil {
			t.Fatalf("outcome: %v", err)
		}
		// Only the OPEN attempt's assist is reported; the concluded grade carries the cap.
		if !res.State.CoachAssistAt.IsZero() {
			t.Fatalf("state after the conclusion still reports coachAssistAt %v", res.State.CoachAssistAt)
		}
		if _, err := st.MarkCoachAssist(ctx, acct, att.ID); !errors.Is(err, store.ErrAttemptClosed) {
			t.Fatalf("assist on a concluded attempt = %v, want ErrAttemptClosed", err)
		}
		if atts, _ := st.ListOpenAttempts(ctx, acct, ""); len(atts) != 0 {
			t.Fatalf("a concluded attempt is still listed open: %+v", atts)
		}
	})

	t.Run("open attempts: account-wide and per problem, newest first", func(t *testing.T) {
		acct := newTestUUID()
		a := openAttempt(t, acct, "1")
		b := openAttempt(t, acct, "2")
		all, err := st.ListOpenAttempts(ctx, acct, "")
		if err != nil || len(all) != 2 || all[0].ID != b.ID || all[1].ID != a.ID {
			t.Fatalf("account-wide = %+v %v, want [%s %s]", all, err, b.ID, a.ID)
		}
		if other, _ := st.ListOpenAttempts(ctx, acct, "3"); len(other) != 0 {
			t.Fatalf("problem 3 = %+v, want none", other)
		}
		if none, _ := st.ListOpenAttempts(ctx, newTestUUID(), ""); len(none) != 0 {
			t.Fatalf("another account sees %+v", none)
		}
	})

	// The clamp matrix: every self-reported outcome × coach used or not, with the hint
	// revealed on the coach rows so `assist.hint` is exercised both ways.
	t.Run("clamp matrix and problem_solved.assist", func(t *testing.T) {
		for _, outcome := range []string{store.OutcomeClean, store.OutcomeRough, store.OutcomeAssisted, store.OutcomeMiss} {
			for _, coach := range []bool{false, true} {
				acct := newTestUUID()
				att := openAttempt(t, acct, "16")
				if coach {
					if _, err := st.Reveal(ctx, acct, "16"); err != nil { // hint
						t.Fatalf("reveal: %v", err)
					}
					if _, err := st.MarkCoachAssist(ctx, acct, att.ID); err != nil {
						t.Fatalf("assist: %v", err)
					}
				}
				res, err := st.LogOutcome(ctx, acct, "16", outcome)
				if err != nil {
					t.Fatalf("%s/coach=%v: outcome: %v", outcome, coach, err)
				}
				capped := coach && (outcome == store.OutcomeClean || outcome == store.OutcomeRough)
				want, wantCap := outcome, ""
				if capped {
					want, wantCap = store.OutcomeAssisted, store.CappedByCoach
				}
				if res.State.LastOutcome != want || res.CappedBy != wantCap {
					t.Errorf("%s/coach=%v: recorded %s cappedBy %q, want %s %q", outcome, coach, res.State.LastOutcome, res.CappedBy, want, wantCap)
				}
				var stored string
				if err := pool.QueryRow(ctx, `SELECT value FROM practice.outcome WHERE attempt_id = $1`, att.ID).Scan(&stored); err != nil || stored != want {
					t.Errorf("%s/coach=%v: outcome row %q %v, want %s", outcome, coach, stored, err, want)
				}

				data := problemSolvedData(t, pool, acct)
				if data.Outcome != want || data.BelowClean != (want != store.OutcomeClean) {
					t.Errorf("%s/coach=%v: problem_solved outcome %s below_clean %v", outcome, coach, data.Outcome, data.BelowClean)
				}
				if data.Assist == nil || data.Assist.Coach != coach || data.Assist.Hint != coach {
					t.Errorf("%s/coach=%v: problem_solved.assist = %+v, want hint=%v coach=%v", outcome, coach, data.Assist, coach, coach)
				}
			}
		}
	})

	// A conclusion racing an assist on the same attempt: the attempt row lock makes
	// them serialise, so either the assist landed first and the grade is capped, or the
	// assist found the attempt closed (the gateway then sends nothing) and it isn't.
	t.Run("assist racing the conclusion", func(t *testing.T) {
		for i := 0; i < 20; i++ {
			acct := newTestUUID()
			att := openAttempt(t, acct, "16")
			var wg sync.WaitGroup
			var res store.OutcomeResult
			var outErr, assistErr error
			wg.Add(2)
			go func() { defer wg.Done(); res, outErr = st.LogOutcome(ctx, acct, "16", store.OutcomeClean) }()
			go func() { defer wg.Done(); _, assistErr = st.MarkCoachAssist(ctx, acct, att.ID) }()
			wg.Wait()
			if outErr != nil {
				t.Fatalf("outcome: %v", outErr)
			}
			switch {
			case assistErr == nil:
				if res.CappedBy != store.CappedByCoach || res.State.LastOutcome != store.OutcomeAssisted {
					t.Fatalf("assist recorded but the outcome is uncapped: %+v", res)
				}
			case errors.Is(assistErr, store.ErrAttemptClosed):
				if res.CappedBy != "" || res.State.LastOutcome != store.OutcomeClean {
					t.Fatalf("assist refused but the outcome is capped: %+v", res)
				}
			default:
				t.Fatalf("assist: %v", assistErr)
			}
		}
	})
}

type solvedData struct {
	Outcome    string `json:"outcome"`
	BelowClean bool   `json:"below_clean"`
	Assist     *struct {
		Hint  bool `json:"hint"`
		Coach bool `json:"coach"`
	} `json:"assist"`
}

// problemSolvedData decodes the account's (single) problem_solved outbox event's data
// with the shared envelope decoder — the one the consumers use.
func problemSolvedData(t *testing.T, pool *pgxpool.Pool, acct string) solvedData {
	t.Helper()
	var payload []byte
	if err := pool.QueryRow(context.Background(),
		`SELECT payload_json FROM practice.outbox WHERE account_id = $1 AND subject = $2`,
		acct, store.SubjectProblemSolved).Scan(&payload); err != nil {
		t.Fatalf("read problem_solved: %v", err)
	}
	env, err := events.DecodeEnvelope(payload)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var d solvedData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		t.Fatalf("data: %v", err)
	}
	return d
}
