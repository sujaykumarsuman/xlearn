// Command assessment is the xLearn assessment service: the timed mock-interview
// aggregate. It runs a 45-minute, server-authoritative phase-rail session (0-5 clarify
// · 5-10 brute force · 10-18 observation->plan · 18-33 code · 33-40 trace+edges ·
// 40-45 complexity+follow-ups), scores it on the 7-dimension rubric (/35), and charts
// the trend vs the readiness targets. Scoring emits xlearn.assessment.mock_completed to
// XLEARN_ASSESSMENT via a transactional outbox (ADR-0004/0005/0006). It also stands up
// durable pull consumers on XLEARN_PRACTICE / XLEARN_REVIEW as the S09 progress-projection
// consume seam (no-op handler bodies this sprint). ClusterIP-only internal service
// (route.enabled: false); the gateway is the only HTTP caller. Schema `assessment` is
// migrated on startup inside a Postgres advisory lock; the service refuses to serve if
// migration fails.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/assessment"
	"github.com/sujaykumarsuman/xlearn/internal/assessment/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
	natsTimeout     = 10 * time.Second
)

func main() { os.Exit(run()) }

func run() int {
	cfg := assessment.LoadConfig()
	logger := slogx.New(cfg.LogLevel)

	// Migrations on startup inside an advisory lock; refuse to serve on failure.
	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancelMigrate()
	if err := store.Migrate(migrateCtx, cfg.DB.DSN(), logger); err != nil {
		logger.Error("migration failed; refusing to serve", "err", err)
		return 1
	}

	pool, err := newPool(context.Background(), cfg.DB)
	if err != nil {
		logger.Error("connect to postgres", "err", err)
		return 1
	}
	defer pool.Close()

	st := store.New(pool)
	verifier := auth.NewJWKSVerifier(cfg.JWT.JWKSURL, cfg.JWT.Audience, cfg.JWT.Issuer)
	svc := assessment.NewService(st, verifier, logger)

	handler := httpx.Chain(svc.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Outbox relay: drains assessment events (mock_completed) to JetStream
	// (XLEARN_ASSESSMENT). With no NATS_URL (local dev) it falls back to the log
	// publisher so the outbox still drains.
	pub := newPublisher(ctx, cfg.NATS.URL, logger)
	if closer, ok := pub.(interface{ Close() }); ok {
		defer closer.Close()
	}
	relay := svc.NewOutboxRelay(pub)
	go relay.Run(ctx)

	// Durable pull consumers for the S09 progress projections: assessment consumes
	// xlearn.practice.* (XLEARN_PRACTICE) and xlearn.review.* (XLEARN_REVIEW). The
	// projection handler bodies are no-op stubs this sprint (S08 wires only the durable
	// + inbox-dedupe seam). With no NATS_URL (local dev) no consumer runs; the HTTP API
	// still works. subs are stopped explicitly on shutdown (before the HTTP drain) so no
	// event is dispatched while draining.
	var subs []events.Subscription
	for _, c := range []struct{ stream, filter string }{
		{assessment.StreamPractice, assessment.PracticeSubjectFilter},
		{assessment.StreamReview, assessment.ReviewSubjectFilter},
	} {
		cons := newConsumer(ctx, cfg.NATS.URL, c.stream, logger)
		if cons == nil {
			continue
		}
		defer cons.Close()
		sub, serr := svc.StartProjectionConsumer(ctx, cons, c.filter)
		if serr != nil {
			logger.Error("start projection consumer", "stream", c.stream, "err", serr)
			return 1
		}
		subs = append(subs, sub)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("assessment listening", "addr", cfg.Addr(), "version", buildVersion())
		if serr := srv.ListenAndServe(); serr != nil && !errors.Is(serr, http.ErrServerClosed) {
			errCh <- serr
		}
	}()

	select {
	case serr := <-errCh:
		logger.Error("server error", "err", serr)
		return 1
	case <-ctx.Done():
		logger.Info("shutdown signal received; draining")
	}

	// Stop consuming BEFORE draining HTTP: halt the Consume loops so no new event is
	// dispatched during shutdown (in-flight handlers run on their own bounded context
	// and finish + ack; the durable consumers resume from their offset on the next
	// start, so nothing is lost).
	for _, sub := range subs {
		sub.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if serr := srv.Shutdown(shutdownCtx); serr != nil {
		logger.Error("graceful shutdown failed", "err", serr)
		return 1
	}
	logger.Info("assessment stopped")
	return 0
}

// newPublisher builds the JetStream publisher when NATS_URL is set, else the log
// publisher (local dev / no broker). A failed NATS connect is non-fatal — the relay
// buffers in the outbox and retries once the broker is reachable.
func newPublisher(ctx context.Context, natsURL string, logger *slog.Logger) events.Publisher {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; using the log publisher (events are not delivered to JetStream)")
		return events.NewLogPublisher(logger, assessment.StreamAssessment)
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	pub, err := events.NewNatsPublisher(natsCtx, natsURL, assessment.StreamAssessment, assessment.StreamSubjects, logger)
	if err != nil {
		logger.Error("nats publisher init failed; falling back to log publisher", "err", err)
		return events.NewLogPublisher(logger, assessment.StreamAssessment)
	}
	return pub
}

// newConsumer builds a durable JetStream consumer on stream when NATS_URL is set, else
// nil (local dev / no broker: the service runs without consuming). A failed connect is
// logged and yields nil; the consumer auto-reconnects once constructed, so Subscribe
// retries the stream.
func newConsumer(ctx context.Context, natsURL, stream string, logger *slog.Logger) *events.NatsConsumer {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; consumers are disabled", "stream", stream)
		return nil
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	cons, err := events.NewNatsConsumer(natsCtx, natsURL, stream, logger)
	if err != nil {
		logger.Error("nats consumer init failed; consumer disabled", "stream", stream, "err", err)
		return nil
	}
	return cons
}

// newPool opens a pgxpool and pins search_path to the service's schema (defence in
// depth; all SQL is schema-qualified anyway).
func newPool(ctx context.Context, db assessment.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(db.DSN())
	if err != nil {
		return nil, err
	}
	if db.SearchPath != "" {
		poolCfg.ConnConfig.RuntimeParams["search_path"] = db.SearchPath
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// buildVersion is stamped via -ldflags at build time (mirrors the other services).
var version = "dev"

func buildVersion() string { return version }
