//go:build e2e

// Package e2e holds the cross-service core-loop end-to-end test. It boots the real
// practice / review / assessment service handlers in-process against a REAL Postgres
// and a REAL NATS JetStream, wired exactly as the cmd/* binaries wire them (transactional
// outbox → relay → JetStream → durable idempotent consumers), and drives events.md
// Flows 1-4 through their HTTP surfaces:
//
//	attempt + clean outcome  →  practice.problem_solved  →  review schedules Day 1/3/7/21/45
//	                                                     →  assessment coverage/mastery projections
//	POST /revisions/{id}/score (pass)                    →  advance the touch ladder
//	below-clean outcome                                  →  review opens a mistake_entry
//	POST /mocks + /mocks/{id}/score                      →  assessment mock stats update Progress
//
// NATS runs IN-PROCESS by default (an embedded nats-server with JetStream), so the test
// is self-contained — it needs only a Postgres — while still exercising the genuine
// broker path; set XLEARN_TEST_NATS_URL to point at an external broker instead. It is
// behind the `e2e` build tag AND gated on XLEARN_TEST_DATABASE_URL, so the default
// `go test ./...` lane never compiles or runs it; CI runs it with `-tags e2e` against a
// Postgres service container (see .github/workflows/ci.yml).
package e2e

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/sujaykumarsuman/xlearn/internal/assessment"
	assessmentstore "github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/practice"
	practicestore "github.com/sujaykumarsuman/xlearn/internal/practice/store"
	"github.com/sujaykumarsuman/xlearn/internal/review"
	reviewstore "github.com/sujaykumarsuman/xlearn/internal/review/store"
)

const (
	issuer      = "xlearn-gateway"
	relayTick   = 150 * time.Millisecond // fast relay so the async loop settles in seconds
	settleFor   = 25 * time.Second       // Eventually budget for an async assertion
	settlePoll  = 150 * time.Millisecond
	problemGood = "16" // solved clean → five-touch ladder + coverage projection
	problemBad  = "18" // below-clean → opens a mistake
)

// TestCoreLoopE2E exercises the whole async method loop over the real broker.
func TestCoreLoopE2E(t *testing.T) {
	dsn := os.Getenv("XLEARN_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set XLEARN_TEST_DATABASE_URL to run the core-loop e2e")
	}
	// NATS is REAL JetStream, in-process by default (self-contained) — or an external
	// broker if XLEARN_TEST_NATS_URL is set. Either way the outbox → relay → JetStream →
	// durable-consumer path is the production one; nothing is mocked.
	natsURL := os.Getenv("XLEARN_TEST_NATS_URL")
	if natsURL == "" {
		natsURL = startEmbeddedNATS(t)
	}

	h := setup(t, dsn, natsURL)
	account := newUUID(t)

	// --- Flow 1: attempt + clean outcome → schedule the five-touch ladder + project coverage.
	h.post(t, h.practiceURL, account, "practice", "/problems/"+problemGood+"/attempt/start", nil, http.StatusOK)
	h.post(t, h.practiceURL, account, "practice", "/problems/"+problemGood+"/outcome",
		[]byte(`{"outcome":"clean"}`), http.StatusOK)

	var due dueQueue
	eventually(t, "review schedules the five-touch ladder", func() bool {
		due = h.dueQueue(t, account)
		return countForProblem(due.Items, problemGood) == 5
	})
	// The ladder is Day 1/3/7/21/45 → touch levels 1..5, exactly once each (idempotent
	// under at-least-once redelivery — no duplicate touches).
	if got := levelsForProblem(due.Items, problemGood); !equalInts(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("touch levels for %s = %v, want [1 2 3 4 5]", problemGood, got)
	}

	eventually(t, "assessment projects the clean solve into Progress coverage", func() bool {
		return h.progressSummary(t, account).Solved >= 1
	})

	// --- Flow 3: score the Day-1 touch (pass) → advance the ladder.
	item := firstForProblem(due.Items, problemGood)
	res := h.scoreRevision(t, account, item.ItemID,
		[]byte(`{"namedPatternSecs":30,"solvedInTimer":true,"statedComplexity":true}`))
	if res.Status != "passed" || !res.AutoPass {
		t.Fatalf("revision score = %+v, want passed auto-pass", res)
	}

	// --- Flow 2: below-clean outcome on another problem → review opens a mistake.
	h.post(t, h.practiceURL, account, "practice", "/problems/"+problemBad+"/attempt/start", nil, http.StatusOK)
	h.post(t, h.practiceURL, account, "practice", "/problems/"+problemBad+"/outcome",
		[]byte(`{"outcome":"miss"}`), http.StatusOK)
	eventually(t, "review opens a mistake_entry on the below-clean outcome", func() bool {
		return h.openMistakes(t, account, problemBad) >= 1
	})

	// --- Mock: start + score a 45-min mock → assessment mock stats reflect it in Progress.
	mockID := h.startMock(t, account)
	h.scoreMock(t, account, mockID)
	eventually(t, "a scored mock lands in Progress mock stats", func() bool {
		return h.progressSummary(t, account).Mock.Count >= 1
	})

	// --- Idempotency / no-double-apply. At-least-once delivery must never duplicate a
	// touch or a projection row. After the settled flow:
	//   - the passed Day-1 touch left EXACTLY four active touches (levels 2-5) for
	//     problem 16 — the score advanced one, and nothing was double-scheduled;
	//   - coverage counts EXACTLY two distinct covered problems (16 clean + 18 miss —
	//     any outcome completes the guided flow, ADR-0018), each once.
	finalDue := h.dueQueue(t, account)
	if got := countForProblem(finalDue.Items, problemGood); got != 4 {
		t.Fatalf("problem %s has %d active touches after the pass, want 4 (advance + no duplication)", problemGood, got)
	}
	// Problem 18's coverage is projected by the assessment consumer asynchronously (a
	// separate durable from review's), and no earlier barrier awaited it — so poll.
	// The exact ==2 is safe under at-least-once: the inbox dedupe makes over-counting
	// impossible, so a genuine double-apply would (correctly) time out here.
	eventually(t, "assessment coverage counts both solved problems exactly (16 + 18, idempotent)", func() bool {
		return h.progressSummary(t, account).Solved == 2
	})
}

// --- harness -----------------------------------------------------------------

type harness struct {
	practiceURL   string
	reviewURL     string
	assessmentURL string
	signer        *auth.Signer
}

func setup(t *testing.T, dsn, natsURL string) *harness {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	// 1. Schemas + migrations. Everything is schema-qualified, so one (superuser) DSN
	//    serves all three; the schemas just have to exist first.
	mustExec(t, dsn, "CREATE SCHEMA IF NOT EXISTS practice",
		"CREATE SCHEMA IF NOT EXISTS review", "CREATE SCHEMA IF NOT EXISTS assessment")
	if err := practicestore.Migrate(ctx, dsn, log); err != nil {
		t.Fatalf("migrate practice: %v", err)
	}
	if err := reviewstore.Migrate(ctx, dsn, log); err != nil {
		t.Fatalf("migrate review: %v", err)
	}
	if err := assessmentstore.Migrate(ctx, dsn, log); err != nil {
		t.Fatalf("migrate assessment: %v", err)
	}

	// 2. A gateway signer + a JWKS endpoint the services verify tokens against.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa: %v", err)
	}
	signer := auth.NewSigner(key, issuer, 30*time.Minute)
	jwks := httptest.NewServer(http.HandlerFunc(auth.JWKSHandler(signer)))
	t.Cleanup(jwks.Close)
	jwksURL := jwks.URL

	// 3. Pools + services (the real handlers, wired as in cmd/*).
	pPool := mustPool(t, ctx, dsn)
	rPool := mustPool(t, ctx, dsn)
	aPool := mustPool(t, ctx, dsn)
	practiceSvc := practice.NewService(practicestore.New(pPool), auth.NewJWKSVerifier(jwksURL, "practice", issuer), log)
	reviewSvc := review.NewService(reviewstore.New(rPool), auth.NewJWKSVerifier(jwksURL, "review", issuer), log)
	assessmentSvc := assessment.NewService(assessmentstore.New(aPool), auth.NewJWKSVerifier(jwksURL, "assessment", issuer), log)

	practiceHTTP := httptest.NewServer(practiceSvc.Handler())
	reviewHTTP := httptest.NewServer(reviewSvc.Handler())
	assessmentHTTP := httptest.NewServer(assessmentSvc.Handler())
	t.Cleanup(practiceHTTP.Close)
	t.Cleanup(reviewHTTP.Close)
	t.Cleanup(assessmentHTTP.Close)

	// 4. Clean-slate the streams so a reused NATS never replays a prior run's history
	//    into this one (DeliverAll consumers would otherwise re-read it). Per-account
	//    isolation would make that harmless, but a clean slate keeps assertions exact.
	deleteStreams(t, ctx, natsURL, practice.StreamPractice, review.StreamReview, assessment.StreamAssessment)

	// 5. Publishers (create the streams) → relays → consumers. Order matters: the
	//    streams must exist before the consumers attach, and the DeliverNew consumers
	//    (assessment, notifications) must be created BEFORE any event is published so
	//    they don't miss the first one.
	pPub := mustPublisher(t, ctx, natsURL, practice.StreamPractice, practice.StreamSubjects, log)
	rPub := mustPublisher(t, ctx, natsURL, review.StreamReview, review.StreamSubjects, log)
	aPub := mustPublisher(t, ctx, natsURL, assessment.StreamAssessment, assessment.StreamSubjects, log)
	go practiceSvc.NewOutboxRelay(pPub, events.WithInterval(relayTick)).Run(ctx)
	go reviewSvc.NewOutboxRelay(rPub, events.WithInterval(relayTick)).Run(ctx)
	go assessmentSvc.NewOutboxRelay(aPub, events.WithInterval(relayTick)).Run(ctx)

	// review consumes XLEARN_PRACTICE (the scheduler) + its own XLEARN_REVIEW (notifications).
	sub, err := reviewSvc.StartConsumers(ctx, mustConsumer(t, ctx, natsURL, review.StreamPractice, log))
	startSub(t, sub, err)
	sub, err = reviewSvc.StartNotifications(ctx, mustConsumer(t, ctx, natsURL, review.StreamReview, log))
	startSub(t, sub, err)
	// assessment projects from XLEARN_PRACTICE + XLEARN_REVIEW.
	sub, err = assessmentSvc.StartProjectionConsumer(ctx, mustConsumer(t, ctx, natsURL, assessment.StreamPractice, log), assessment.PracticeSubjectFilter)
	startSub(t, sub, err)
	sub, err = assessmentSvc.StartProjectionConsumer(ctx, mustConsumer(t, ctx, natsURL, assessment.StreamReview, log), assessment.ReviewSubjectFilter)
	startSub(t, sub, err)

	return &harness{practiceURL: practiceHTTP.URL, reviewURL: reviewHTTP.URL, assessmentURL: assessmentHTTP.URL, signer: signer}
}

// --- HTTP helpers ------------------------------------------------------------

func (h *harness) mint(t *testing.T, account, audience string) string {
	t.Helper()
	tok, err := h.signer.Mint(context.Background(), account, audience, []string{"learner"})
	if err != nil {
		t.Fatalf("mint %s: %v", audience, err)
	}
	return tok
}

func (h *harness) post(t *testing.T, base, account, aud, path string, body []byte, wantStatus int) []byte {
	t.Helper()
	return h.do(t, http.MethodPost, base, account, aud, path, body, wantStatus)
}

func (h *harness) get(t *testing.T, base, account, aud, path string) []byte {
	t.Helper()
	return h.do(t, http.MethodGet, base, account, aud, path, nil, http.StatusOK)
}

func (h *harness) do(t *testing.T, method, base, account, aud, path string, body []byte, wantStatus int) []byte {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, base+path, rdr)
	if err != nil {
		t.Fatalf("build %s %s: %v", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+h.mint(t, account, aud))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s = %d (%s), want %d", method, path, resp.StatusCode, out, wantStatus)
	}
	return out
}

type dueItem struct {
	ItemID     string `json:"itemId"`
	ProblemID  string `json:"problemId"`
	TouchLevel int    `json:"touchLevel"`
	Status     string `json:"status"`
}
type dueQueue struct {
	Items []dueItem `json:"items"`
}

func (h *harness) dueQueue(t *testing.T, account string) dueQueue {
	t.Helper()
	var q dueQueue
	mustJSON(t, h.get(t, h.reviewURL, account, "review", "/revisions/due"), &q)
	return q
}

type scoreResult struct {
	Status   string `json:"status"`
	AutoPass bool   `json:"autoPass"`
}

func (h *harness) scoreRevision(t *testing.T, account, itemID string, body []byte) scoreResult {
	t.Helper()
	var r scoreResult
	mustJSON(t, h.post(t, h.reviewURL, account, "review", "/revisions/"+itemID+"/score", body, http.StatusOK), &r)
	return r
}

func (h *harness) openMistakes(t *testing.T, account, problemID string) int {
	t.Helper()
	var env struct {
		Mistakes []struct {
			ProblemID string `json:"problemId"`
			Status    string `json:"status"`
		} `json:"mistakes"`
	}
	mustJSON(t, h.get(t, h.reviewURL, account, "review", "/mistakes?status=open"), &env)
	n := 0
	for _, m := range env.Mistakes {
		if m.ProblemID == problemID && m.Status == "open" {
			n++
		}
	}
	return n
}

type summary struct {
	Solved int `json:"solved"`
	Mock   struct {
		Count int `json:"count"`
	} `json:"mock"`
}

func (h *harness) progressSummary(t *testing.T, account string) summary {
	t.Helper()
	var s summary
	mustJSON(t, h.get(t, h.assessmentURL, account, "assessment", "/progress/summary"), &s)
	return s
}

func (h *harness) startMock(t *testing.T, account string) string {
	t.Helper()
	var v struct {
		ID string `json:"id"`
	}
	mustJSON(t, h.post(t, h.assessmentURL, account, "assessment", "/mocks",
		[]byte(`{"setId":"w13","problemId":"`+problemGood+`","difficulty":"med"}`), http.StatusCreated), &v)
	if v.ID == "" {
		t.Fatal("start mock returned no id")
	}
	return v.ID
}

func (h *harness) scoreMock(t *testing.T, account, mockID string) {
	t.Helper()
	body := []byte(`{"scores":{"communication":4,"problem_understanding":4,"brute_force":3,` +
		`"optimisation":3,"code_quality":4,"edge_cases":3,"complexity":3}}`)
	h.post(t, h.assessmentURL, account, "assessment", "/mocks/"+mockID+"/score", body, http.StatusOK)
}

// --- low-level helpers -------------------------------------------------------

// startEmbeddedNATS runs a real in-process NATS JetStream server (file store under a
// temp dir) and returns its client URL. This keeps the e2e self-contained — it needs
// only a Postgres — while still exercising the genuine JetStream + outbox path.
func startEmbeddedNATS(t *testing.T) string {
	t.Helper()
	ns, err := natsserver.NewServer(&natsserver.Options{
		Host:      "127.0.0.1",
		Port:      -1, // an ephemeral free port
		JetStream: true,
		StoreDir:  t.TempDir(),
		NoLog:     true,
		NoSigs:    true,
	})
	if err != nil {
		t.Fatalf("embedded nats: %v", err)
	}
	go ns.Start()
	if !ns.ReadyForConnections(15 * time.Second) {
		t.Fatal("embedded nats not ready")
	}
	t.Cleanup(ns.Shutdown)
	return ns.ClientURL()
}

func mustPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func mustExec(t *testing.T, dsn string, stmts ...string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("exec pool: %v", err)
	}
	defer conn.Close()
	for _, s := range stmts {
		if _, err := conn.Exec(ctx, s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
}

func mustPublisher(t *testing.T, ctx context.Context, url, stream string, subjects []string, log *slog.Logger) *events.NatsPublisher {
	t.Helper()
	pub, err := events.NewNatsPublisher(ctx, url, stream, subjects, log)
	if err != nil {
		t.Fatalf("publisher %s: %v", stream, err)
	}
	t.Cleanup(pub.Close)
	return pub
}

func mustConsumer(t *testing.T, ctx context.Context, url, stream string, log *slog.Logger) *events.NatsConsumer {
	t.Helper()
	cons, err := events.NewNatsConsumer(ctx, url, stream, log)
	if err != nil {
		t.Fatalf("consumer %s: %v", stream, err)
	}
	t.Cleanup(cons.Close)
	return cons
}

func startSub(t *testing.T, sub events.Subscription, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("start consumer: %v", err)
	}
	t.Cleanup(sub.Stop)
}

// deleteStreams removes the JetStream streams for a clean-slate run (ignoring
// not-found), so a reused broker never leaks a prior run's events into this one.
func deleteStreams(t *testing.T, ctx context.Context, url string, names ...string) {
	t.Helper()
	nc, err := nats.Connect(url)
	if err != nil {
		t.Fatalf("nats connect for cleanup: %v", err)
	}
	defer nc.Close()
	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream for cleanup: %v", err)
	}
	dctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for _, n := range names {
		if derr := js.DeleteStream(dctx, n); derr != nil && !errors.Is(derr, jetstream.ErrStreamNotFound) {
			t.Logf("delete stream %s: %v (continuing)", n, derr)
		}
	}
}

func mustJSON(t *testing.T, body []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
}

// eventually polls fn until it returns true or the settle budget elapses.
func eventually(t *testing.T, what string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(settleFor)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(settlePoll)
	}
	t.Fatalf("timed out after %s waiting for: %s", settleFor, what)
}

func countForProblem(items []dueItem, pid string) int {
	n := 0
	for _, it := range items {
		if it.ProblemID == pid && it.Status != "passed" {
			n++
		}
	}
	return n
}

func levelsForProblem(items []dueItem, pid string) []int {
	var out []int
	for _, it := range items {
		if it.ProblemID == pid {
			out = append(out, it.TouchLevel)
		}
	}
	sortInts(out)
	return out
}

func firstForProblem(items []dueItem, pid string) dueItem {
	for _, it := range items {
		if it.ProblemID == pid {
			return it
		}
	}
	return dueItem{}
}

func sortInts(a []int) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// newUUID returns a random RFC-4122 v4 UUID string (account ids are uuid-typed).
func newUUID(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatalf("uuid: %v", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
