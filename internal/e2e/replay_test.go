//go:build e2e

package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment"
	assessmentstore "github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/review"
	reviewstore "github.com/sujaykumarsuman/xlearn/internal/review/store"
)

// TestReplayV1EventsAndV2Twin is m1-02's replay check (and the fixture m1-08's contract
// rehearsal reuses): internal/e2e/testdata/v1-events.jsonl holds 19 SYNTHETIC v1
// envelopes shaped like production's event log (one account; practice solves, attempts
// and an early reveal; review schedules, mistakes and a due). They are replayed, in
// order, through the REAL review (practice + notifications) and assessment (projection)
// handlers — captured from the services' own Start* wiring, so the code path is the
// production one minus the broker — onto a fresh schema. The resulting review rows,
// review outbox facts and assessment projections must equal the golden
// (testdata/v1-events.golden.json; XLEARN_UPDATE_GOLDEN=1 rewrites it), and the same 19
// events re-encoded as v2 envelopes (path_slug "dsa", another account) must produce an
// identical snapshot. A second delivery of every event changes nothing (inbox dedupe).
// Since m1-03 review's own outbox facts are v2 envelopes (checked beside the golden,
// which compares only their data and so stays v1's).
func TestReplayV1EventsAndV2Twin(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the replay e2e")
	}
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db := freshDatabase(t, dsn, "replay")
	mustExec(t, db, "CREATE SCHEMA review", "CREATE SCHEMA assessment")
	if err := reviewstore.Migrate(ctx, db, log); err != nil {
		t.Fatalf("migrate review: %v", err)
	}
	if err := assessmentstore.Migrate(ctx, db, log); err != nil {
		t.Fatalf("migrate assessment: %v", err)
	}
	pool := mustPool(t, ctx, db)
	aStore := assessmentstore.New(pool)
	noJWKS := "http://127.0.0.1:1/jwks" // the handlers under test never verify a token
	reviewSvc := review.NewService(reviewstore.New(pool), auth.NewJWKSVerifier(noJWKS, "review", issuer), log)
	assessmentSvc := assessment.NewService(aStore, auth.NewJWKSVerifier(noJWKS, "assessment", issuer), log)

	// Capture every durable handler exactly as the services bind them.
	capt := &captureConsumer{}
	for _, err := range []error{
		second(reviewSvc.StartConsumers(ctx, capt)),
		second(reviewSvc.StartNotifications(ctx, capt)),
		second(assessmentSvc.StartProjectionConsumer(ctx, capt, assessment.PracticeSubjectFilter)),
		second(assessmentSvc.StartProjectionConsumer(ctx, capt, assessment.ReviewSubjectFilter)),
	} {
		if err != nil {
			t.Fatalf("bind handlers: %v", err)
		}
	}
	if len(capt.subs) != 4 {
		t.Fatalf("%d durable handlers bound, want 4", len(capt.subs))
	}

	v1 := readReplayFixture(t)
	if len(v1) != 19 {
		t.Fatalf("fixture has %d events, want 19", len(v1))
	}
	accountV1 := v1[0]["account_id"].(string)
	accountV2 := "5f0c2a1e-7b3d-4c8e-9a6f-2222222222c2"
	v2 := make([]map[string]any, len(v1))
	for i, e := range v1 {
		v2[i] = reencodeV2(e, accountV2)
	}

	capt.replay(t, ctx, v1)
	capt.replay(t, ctx, v2)
	got := replaySnapshot(t, ctx, pool, aStore, accountV1)
	twin := replaySnapshot(t, ctx, pool, aStore, accountV2)

	// Redelivery of every event is a no-op (the inbox dedupes on event_id).
	capt.replay(t, ctx, v1)
	if again := replaySnapshot(t, ctx, pool, aStore, accountV1); again != got {
		t.Fatalf("a second delivery changed the projections:\n%s\nvs\n%s", again, got)
	}

	if twin != got {
		t.Fatalf("the v2 twin differs from the v1 replay:\nv1: %s\nv2: %s", got, twin)
	}

	// m1-03: review's producers emit v2. Every outbox fact the replay produced (for
	// either input version) is a v2 envelope in the DSA course; the snapshot compares
	// only the facts' data, which stays v1's (golden = v1).
	var facts, v2Facts int
	if err := pool.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE payload_json ->> 'version' = '2' AND payload_json ->> 'path_slug' = $1)
		FROM review.outbox`, events.V1PathSlug).Scan(&facts, &v2Facts); err != nil {
		t.Fatalf("read review outbox envelopes: %v", err)
	}
	if facts == 0 || v2Facts != facts {
		t.Fatalf("%d review outbox facts, %d of them v2 in the DSA course; want all", facts, v2Facts)
	}
	golden := filepath.Join("testdata", "v1-events.golden.json")
	if os.Getenv("XLEARN_UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (XLEARN_UPDATE_GOLDEN=1 writes it): %v", err)
	}
	if string(want) != got {
		t.Fatalf("replay differs from %s (golden = v1):\ngot  %s\nwant %s", golden, got, want)
	}
}

// captureConsumer is an events.Consumer that records each bound handler instead of
// subscribing, so the replay drives the production handlers synchronously.
type captureConsumer struct {
	subs []capturedSub
}

type capturedSub struct {
	durable, filter string
	h               events.Handler
}

func (c *captureConsumer) Subscribe(_ context.Context, durable, filter string, h events.Handler, _ ...events.SubscribeOption) (events.Subscription, error) {
	c.subs = append(c.subs, capturedSub{durable, filter, h})
	return noopSub{}, nil
}

type noopSub struct{}

func (noopSub) Stop() {}

// replay delivers each event to every captured handler whose filter captures its
// subject — what JetStream does for the durables — and fails on any handler error.
func (c *captureConsumer) replay(t *testing.T, ctx context.Context, evs []map[string]any) {
	t.Helper()
	for _, e := range evs {
		data, err := json.Marshal(e)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		subject := e["subject"].(string)
		for _, s := range c.subs {
			if !events.SubjectMatches(s.filter, subject) {
				continue
			}
			if err := s.h.Handle(ctx, events.Event{ID: e["event_id"].(string), Subject: subject, Data: data}); err != nil {
				t.Fatalf("%s handler on %s (%s): %v", s.durable, subject, e["event_id"], err)
			}
		}
	}
}

func readReplayFixture(t *testing.T) []map[string]any {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "v1-events.jsonl"))
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()
	var out []map[string]any
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e map[string]any
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatalf("fixture line: %v", err)
		}
		if e["version"].(float64) != 1 {
			t.Fatalf("fixture event %v is not v1", e["event_id"])
		}
		out = append(out, e)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan fixture: %v", err)
	}
	return out
}

// reencodeV2 turns a v1 fixture event into its v2 twin for another account: version 2,
// path_slug "dsa", and an event id of its own (so the inbox doesn't dedupe it away).
func reencodeV2(e map[string]any, account string) map[string]any {
	out := map[string]any{}
	for k, v := range e {
		out[k] = v
	}
	out["version"] = 2
	out["path_slug"] = "dsa"
	out["account_id"] = account
	out["event_id"] = "e2" + e["event_id"].(string)[2:]
	return out
}

// replaySnapshot renders an account's review rows, review outbox facts and assessment
// projections as canonical JSON, with the account id, row ids and wall-clock stamps
// left out, so two accounts' snapshots compare directly.
func replaySnapshot(t *testing.T, ctx context.Context, pool *pgxpool.Pool, st *assessmentstore.PgStore, account string) string {
	t.Helper()
	rows := func(q string) []string {
		t.Helper()
		r, err := pool.Query(ctx, q, account)
		if err != nil {
			t.Fatalf("query %q: %v", q, err)
		}
		defer r.Close()
		var out []string
		for r.Next() {
			var s string
			if err := r.Scan(&s); err != nil {
				t.Fatalf("scan: %v", err)
			}
			out = append(out, s)
		}
		sort.Strings(out)
		return out
	}
	snap := map[string]any{
		"revision_item": rows(`SELECT concat_ws('|', problem_id, touch_level, to_char(due_date AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), status, path_slug)
			FROM review.revision_item WHERE account_id = $1`),
		"mistake_entry": rows(`SELECT concat_ws('|', problem_id, status, coalesce(category, '-'), pattern, revisit_count, path_slug)
			FROM review.mistake_entry WHERE account_id = $1`),
		"reminder": rows(`SELECT concat_ws('|', kind, coalesce(path_slug, '-'))
			FROM review.reminder WHERE account_id = $1`),
		"review_outbox": rows(`SELECT concat_ws('|', subject, (payload_json -> 'data')::text, account_id IS NOT NULL)
			FROM review.outbox WHERE payload_json ->> 'account_id' = $1::text`),
	}
	solved, err := st.SolvedCount(ctx, account)
	must(t, err)
	ladders, resets, err := st.Retention(ctx, account)
	must(t, err)
	heat, err := st.Heatmap(ctx, account, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	must(t, err)
	mastery, err := st.Mastery(ctx, account)
	must(t, err)
	mix, err := st.OutcomeMix(ctx, account)
	must(t, err)
	var heatRows []string
	for _, d := range heat {
		d.Date = d.Date.UTC() // times in UTC: the golden must not depend on the machine's zone
		heatRows = append(heatRows, fmt.Sprintf("%+v", d))
	}
	var masteryRows []string
	for _, m := range mastery {
		first := "-"
		if m.FirstSolvedAt != nil {
			first = m.FirstSolvedAt.UTC().Format(time.RFC3339)
		}
		m.FirstSolvedAt = nil
		masteryRows = append(masteryRows, fmt.Sprintf("%s|%+v", first, m))
	}
	sort.Strings(heatRows)
	sort.Strings(masteryRows)
	snap["assessment"] = map[string]any{
		"solved": solved, "ladders": ladders, "resets": resets,
		"heatmap": heatRows, "mastery": masteryRows, "outcome_mix": mix,
	}
	b, err := json.MarshalIndent(snap, "", "  ")
	must(t, err)
	return strings.TrimSpace(string(b)) + "\n"
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func second[T any](_ T, err error) error { return err }

// freshDatabase creates a throwaway database on dsn's server (superuser DSN) and returns
// a DSN to it; it is dropped when the test ends.
func freshDatabase(t *testing.T, dsn, prefix string) string {
	t.Helper()
	name := fmt.Sprintf("m1a_%s_%d", prefix, time.Now().UnixNano())
	mustExec(t, dsn, "CREATE DATABASE "+name)
	t.Cleanup(func() {
		ctx := context.Background()
		p, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return
		}
		defer p.Close()
		_, _ = p.Exec(ctx, "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatalf("parse dsn: %v", err)
	}
	u.Path = "/" + name
	return u.String()
}
