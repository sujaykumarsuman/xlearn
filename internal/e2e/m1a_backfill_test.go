//go:build e2e

package e2e

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"

	assessmentstore "github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	coachstore "github.com/sujaykumarsuman/xlearn/internal/coach/store"
	identitystore "github.com/sujaykumarsuman/xlearn/internal/identity/store"
	practicestore "github.com/sujaykumarsuman/xlearn/internal/practice/store"
	reviewstore "github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// The goose versions each schema had at v1.5.2 (hack/migrations-baseline.txt).
const (
	v152Practice   = 1
	v152Review     = 3
	v152Assessment = 2
	v152Coach      = 3
	v152Identity   = 4
)

// TestM1aBackfillOnV152Schema is m1-02's backfill rehearsal: build every schema at its
// v1.5.2 version, seed it the way v1.5.2 writes (SYNTHETIC rows: two accounts, a scored
// mock and a mixed-set live one, two coach keys, …), run the M1a expand migrations, and
// check every backfill: path_slug 'dsa', the attempt's denormalised ids,
// outbox.account_id, total = total_35 / max_total 35 / scored_by 'self' / the rubric,
// one mock_session_item per session (item_id NULL for a ” session), the relaxed CHECK,
// key_default = the is_default rows, and admitted_via 'grandfathered' for every account.
// The compose rehearsal (real v1.5.2 images, then this branch, then v1.5.2 again) is
// the end-to-end twin of this test.
func TestM1aBackfillOnV152Schema(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the M1a backfill rehearsal")
	}
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := freshDatabase(t, dsn, "backfill")
	mustExec(t, db, "CREATE SCHEMA practice", "CREATE SCHEMA review", "CREATE SCHEMA assessment",
		"CREATE SCHEMA coach", "CREATE SCHEMA identity")

	type mig struct {
		name     string
		to       func(context.Context, string, int64, *slog.Logger) error
		all      func(context.Context, string, *slog.Logger) error
		baseline int64
	}
	migs := []mig{
		{"practice", practicestore.MigrateTo, practicestore.Migrate, v152Practice},
		{"review", reviewstore.MigrateTo, reviewstore.Migrate, v152Review},
		{"assessment", assessmentstore.MigrateTo, assessmentstore.Migrate, v152Assessment},
		{"coach", coachstore.MigrateTo, coachstore.Migrate, v152Coach},
		{"identity", identitystore.MigrateTo, identitystore.Migrate, v152Identity},
	}
	for _, m := range migs {
		if err := m.to(ctx, db, m.baseline, log); err != nil {
			t.Fatalf("migrate %s to v1.5.2 (%d): %v", m.name, m.baseline, err)
		}
	}

	const (
		owner = "6a1b2c3d-4e5f-4a6b-8c7d-000000000001"
		other = "6a1b2c3d-4e5f-4a6b-8c7d-000000000002"
	)
	// v1.5.2-shaped rows, written with the columns v1.5.2 knows.
	mustExec(t, db,
		`INSERT INTO identity.account (id, display_name, email) VALUES ('`+owner+`', 'Owner', 'owner@example.test'), ('`+other+`', 'Other', NULL)`,
		`INSERT INTO identity.path_enrollment (account_id, path_slug) VALUES ('`+owner+`', 'dsa')`,

		`INSERT INTO practice.user_problem_state (id, account_id, problem_id, status) VALUES ('7a000000-0000-4000-8000-000000000001', '`+owner+`', '16', 'solved')`,
		`INSERT INTO practice.attempt (user_problem_state_id) VALUES ('7a000000-0000-4000-8000-000000000001')`,
		`INSERT INTO practice.outbox (event_id, subject, payload_json) VALUES
			('7b000000-0000-4000-8000-000000000001', 'xlearn.practice.problem_solved', '{"account_id": "`+owner+`", "version": 1}'),
			('7b000000-0000-4000-8000-000000000002', 'xlearn.practice.problem_solved', '{"account_id": "not-a-uuid", "version": 1}')`,

		`INSERT INTO review.revision_item (account_id, problem_id, touch_level, due_date) VALUES ('`+owner+`', '16', 1, now())`,
		`INSERT INTO review.mistake_entry (account_id, problem_id) VALUES ('`+owner+`', '18')`,
		`INSERT INTO review.weak_area_snapshot (account_id, week_of) VALUES ('`+owner+`', '2026-09-21')`,
		`INSERT INTO review.reminder (account_id, kind, due_at) VALUES ('`+owner+`', 'revision_due', now())`,
		`INSERT INTO review.outbox (event_id, subject, payload_json) VALUES ('7c000000-0000-4000-8000-000000000001', 'xlearn.review.revision_scheduled', '{"account_id": "`+owner+`"}')`,

		`INSERT INTO assessment.mock_session (id, account_id, set_id, problem_id, difficulty, status, total_35, deadline_at) VALUES
			('7d000000-0000-4000-8000-000000000001', '`+owner+`', 'w13', '16', 'med', 'scored', 24, now()),
			('7d000000-0000-4000-8000-000000000002', '`+owner+`', 'w14', '', 'hard', 'live', NULL, now())`,
		`INSERT INTO assessment.rubric_score (mock_session_id, dimension, score) VALUES ('7d000000-0000-4000-8000-000000000001', 'communication', 4)`,
		`INSERT INTO assessment.outbox (event_id, subject, payload_json) VALUES ('7e000000-0000-4000-8000-000000000001', 'xlearn.assessment.mock_completed', '{"account_id": "`+owner+`"}')`,

		`INSERT INTO coach.api_key_config (id, account_id, provider, enc_key, enc_data_key, masked_key, default_model, is_default) VALUES
			('7f000000-0000-4000-8000-000000000001', '`+owner+`', 'openai', '\x01', '\x02', 'sk-…1', 'gpt-x', true),
			('7f000000-0000-4000-8000-000000000002', '`+owner+`', 'anthropic', '\x01', '\x02', 'sk-…2', 'claude-x', false),
			('7f000000-0000-4000-8000-000000000003', '`+other+`', 'anthropic', '\x01', '\x02', 'sk-…3', 'claude-y', true)`,
		`INSERT INTO coach.coach_thread (account_id, page_context) VALUES ('`+owner+`', 'problem:16')`,
	)

	// The M1a expand, as a v1.6.0 service's startup runs it (twice: idempotent).
	for i := 0; i < 2; i++ {
		for _, m := range migs {
			if err := m.all(ctx, db, log); err != nil {
				t.Fatalf("migrate %s (run %d): %v", m.name, i+1, err)
			}
		}
	}

	pool := mustPool(t, ctx, db)
	expect := func(what, q string, want any) {
		t.Helper()
		var got any
		if err := pool.QueryRow(ctx, q).Scan(&got); err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if got != want {
			t.Fatalf("%s = %v (%T), want %v", what, got, got, want)
		}
	}

	// identity
	expect("accounts not grandfathered", `SELECT count(*) FROM identity.account WHERE admitted_via IS DISTINCT FROM 'grandfathered'`, int64(0))
	expect("accounts with non-default role/status/visibility",
		`SELECT count(*) FROM identity.account WHERE role <> 'learner' OR status <> 'active' OR profile_visibility <> 'public'`, int64(0))
	expect("enrollment public_visible", `SELECT public_visible FROM identity.path_enrollment WHERE account_id = '`+owner+`'`, true)
	expect("admin_audit rows", `SELECT count(*) FROM identity.admin_audit`, int64(0))

	// practice
	expect("user_problem_state.path_slug", `SELECT path_slug FROM practice.user_problem_state`, "dsa")
	expect("attempts not denormalised", `SELECT count(*) FROM practice.attempt a JOIN practice.user_problem_state s ON s.id = a.user_problem_state_id
		WHERE a.account_id IS DISTINCT FROM s.account_id OR a.path_slug IS DISTINCT FROM s.path_slug OR a.problem_id IS DISTINCT FROM s.problem_id`, int64(0))
	expect("practice outbox backfilled", `SELECT count(*) FROM practice.outbox WHERE account_id = '`+owner+`'`, int64(1))
	expect("malformed payload left NULL", `SELECT count(*) FROM practice.outbox WHERE account_id IS NULL`, int64(1))

	// review
	for _, tbl := range []string{"revision_item", "mistake_entry", "weak_area_snapshot"} {
		expect("review."+tbl+" path_slug", `SELECT path_slug FROM review.`+tbl, "dsa")
	}
	expect("reminder.path_slug stays NULL", `SELECT count(*) FROM review.reminder WHERE path_slug IS NULL`, int64(1))
	expect("review outbox backfilled", `SELECT count(*) FROM review.outbox WHERE account_id = '`+owner+`'`, int64(1))
	expect("weak-area unique valid", `SELECT i.indisvalid FROM pg_index i JOIN pg_class c ON c.oid = i.indexrelid
		WHERE c.relname = 'weak_area_snapshot_account_path_week_uq'`, true)

	// assessment
	expect("scored mock totals", `SELECT concat_ws('|', total, max_total, scored_by, rubric_id, path_slug, rubric_snapshot = '`+assessmentstore.RubricSnapshot+`'::jsonb)
		FROM assessment.mock_session WHERE id = '7d000000-0000-4000-8000-000000000001'`, "24|35|self|dsa-mock@1|dsa|t")
	expect("live mock (no total, rubric set)", `SELECT concat_ws('|', coalesce(total::text, '-'), max_total, coalesce(scored_by, '-'), rubric_id)
		FROM assessment.mock_session WHERE id = '7d000000-0000-4000-8000-000000000002'`, "-|35|-|dsa-mock@1")
	expect("mock_session_item rows", `SELECT string_agg(concat_ws('|', ordinal, coalesce(item_id, 'NULL'), path_slug), ',' ORDER BY session_id)
		FROM assessment.mock_session_item`, "1|16|dsa,1|NULL|dsa")
	expect("old mock CHECK dropped, new one validated", `SELECT string_agg(conname || ':' || convalidated::text, ',' ORDER BY conname)
		FROM pg_constraint WHERE conrelid = 'assessment.mock_session'::regclass AND conname IN ('mock_session_check', 'mock_session_scored_total_check')`,
		"mock_session_scored_total_check:true")
	expect("assessment outbox backfilled", `SELECT count(*) FROM assessment.outbox WHERE account_id = '`+owner+`'`, int64(1))

	// coach
	expect("key_default = the is_default rows", `SELECT string_agg(concat_ws('|', d.account_id, k.provider, d.model, d.feature), ',' ORDER BY d.account_id)
		FROM coach.key_default d JOIN coach.api_key_config k ON k.id = d.key_id`,
		owner+"|openai|gpt-x|coach,"+other+"|anthropic|claude-y|coach")
	expect("is_default nullable", `SELECT is_nullable FROM information_schema.columns WHERE table_schema = 'coach' AND table_name = 'api_key_config' AND column_name = 'is_default'`, "YES")
	expect("coach path_slug nullable, unset", `SELECT count(*) FROM coach.coach_thread WHERE path_slug IS NULL`, int64(1))
}
