// Command review is the xLearn review service: the five-touch spaced-repetition
// scheduler and the Revision queue — the first async CONSUMER in the system. It
// subscribes durable pull consumers to XLEARN_PRACTICE (xlearn.practice.*) to
// schedule the Day 1/3/7/21/45 ladder on a first clean solve and the owed 3-day
// re-solve on an early reveal, auto-scores re-solves (advance or reset-to-Day-1), and
// runs a ~15-minute sweep that materialises "due today" (offline-safe). It emits
// revision_scheduled / revision_due to XLEARN_REVIEW via a transactional outbox
// (ADR-0004/0005/0006). ClusterIP-only internal service (route.enabled: false); the
// gateway is the only HTTP caller. Schema `review` is migrated on startup inside a
// Postgres advisory lock; the service refuses to serve if migration fails.
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

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
	"github.com/sujaykumarsuman/xlearn/internal/review"
	"github.com/sujaykumarsuman/xlearn/internal/review/store"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
	natsTimeout     = 10 * time.Second
)

func main() { os.Exit(run()) }

func run() int {
	cfg := review.LoadConfig()
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
	svc := review.NewService(st, verifier, logger)

	handler := httpx.Chain(svc.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Outbox relay: drains review events to JetStream (XLEARN_REVIEW). With no
	// NATS_URL (local dev) it falls back to the log publisher so the outbox drains.
	pub := newPublisher(ctx, cfg.NATS.URL, logger)
	if closer, ok := pub.(interface{ Close() }); ok {
		defer closer.Close()
	}
	relay := svc.NewOutboxRelay(pub)
	go relay.Run(ctx)

	// Durable pull consumer on XLEARN_PRACTICE (xlearn.practice.*) — the five-touch
	// scheduler's event source. With no NATS_URL (local dev) there is no broker, so
	// no consumer runs; the HTTP API + sweep still work. sub is stopped explicitly on
	// shutdown (before the HTTP drain) so no event is dispatched while draining.
	var sub events.Subscription
	if cons := newConsumer(ctx, cfg.NATS.URL, logger); cons != nil {
		defer cons.Close()
		s, serr := svc.StartConsumers(ctx, cons)
		if serr != nil {
			logger.Error("start practice consumer", "err", serr)
			return 1
		}
		sub = s
	}

	// Periodic sweep: materialises due-today reliably even after days offline (R-SR6).
	sweeper := svc.NewSweeper(cfg.Sweep.Interval, cfg.Sweep.Batch)
	go sweeper.Run(ctx)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("review listening", "addr", cfg.Addr(), "version", buildVersion())
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

	// Stop consuming BEFORE draining HTTP: halt the Consume loop so no new practice
	// event is dispatched during shutdown (in-flight handlers run on their own bounded
	// context and finish + ack; the durable consumer resumes from its offset on the
	// next start, so nothing is lost).
	if sub != nil {
		sub.Stop()
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if serr := srv.Shutdown(shutdownCtx); serr != nil {
		logger.Error("graceful shutdown failed", "err", serr)
		return 1
	}
	logger.Info("review stopped")
	return 0
}

// newPublisher builds the JetStream publisher when NATS_URL is set, else the log
// publisher (local dev / no broker). A failed NATS connect is non-fatal — the relay
// buffers in the outbox and retries once the broker is reachable.
func newPublisher(ctx context.Context, natsURL string, logger *slog.Logger) events.Publisher {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; using the log publisher (events are not delivered to JetStream)")
		return events.NewLogPublisher(logger, review.StreamReview)
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	pub, err := events.NewNatsPublisher(natsCtx, natsURL, review.StreamReview, review.StreamSubjects, logger)
	if err != nil {
		logger.Error("nats publisher init failed; falling back to log publisher", "err", err)
		return events.NewLogPublisher(logger, review.StreamReview)
	}
	return pub
}

// newConsumer builds the durable JetStream consumer when NATS_URL is set, else nil
// (local dev / no broker: the service runs without consuming). A failed connect is
// fatal only if NATS_URL was set but unreachable at construction — but the consumer
// auto-reconnects, so construction succeeds and Subscribe retries the stream.
func newConsumer(ctx context.Context, natsURL string, logger *slog.Logger) *events.NatsConsumer {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; the practice consumer is disabled (no five-touch scheduling from events)")
		return nil
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	cons, err := events.NewNatsConsumer(natsCtx, natsURL, review.StreamPractice, logger)
	if err != nil {
		logger.Error("nats consumer init failed; the practice consumer is disabled", "err", err)
		return nil
	}
	return cons
}

// newPool opens a pgxpool and pins search_path to the service's schema (defence in
// depth; all SQL is schema-qualified anyway).
func newPool(ctx context.Context, db review.DBConfig) (*pgxpool.Pool, error) {
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
