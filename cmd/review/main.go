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
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/config"
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

func main() {
	// `review -version` prints the -ldflags-stamped version and exits (guarded by
	// deploy/version_test.go).
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Println(buildVersion())
		return
	}
	os.Exit(run())
}

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

	// Internal clients for soft cross-context references (ADR-0005/0016): curriculum
	// pre-fills a mistake's pattern; identity resolves account timezone/study-budget for
	// the background workers. Both are nil when their base URL is unset (local dev),
	// which the store + workers degrade around (empty pattern / UTC / immediate).
	var storeOpts []store.Option
	if curriculum := review.NewCurriculumClient(cfg.CurriculumBaseURL); curriculum != nil {
		storeOpts = append(storeOpts, store.WithPatternResolver(curriculum))
	}
	st := store.New(pool, storeOpts...)
	verifier := auth.NewJWKSVerifier(cfg.JWT.JWKSURL, cfg.JWT.Audience, cfg.JWT.Issuer)
	svc := review.NewService(st, verifier, logger)
	if identity := review.NewIdentityClient(cfg.IdentityBaseURL); identity != nil {
		svc.WithAccountResolver(identity)
	}

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
	// With NATS_URL set an init error is fatal (fail closed): the log publisher would
	// mark rows sent without delivering them.
	pub, err := newPublisher(ctx, cfg.NATS.URL, logger)
	if err != nil {
		logger.Error("nats publisher init failed; refusing to start (NATS_URL is set)", "err", err)
		return 1
	}
	if closer, ok := pub.(interface{ Close() }); ok {
		defer closer.Close()
	}
	relay := svc.NewOutboxRelay(pub)
	go relay.Run(ctx)

	// Durable pull consumer on XLEARN_PRACTICE (xlearn.practice.*) — the five-touch
	// scheduler's event source. With no NATS_URL (local dev) there is no broker, so
	// no consumer runs; the HTTP API + sweep still work. With NATS_URL set a consumer
	// init error is fatal (fail closed). sub is stopped explicitly on shutdown (before
	// the HTTP drain) so no event is dispatched while draining.
	var subs []events.Subscription
	practiceCons, err := newConsumer(ctx, cfg.NATS.URL, review.StreamPractice, logger)
	if err != nil {
		logger.Error("nats consumer init failed; refusing to start (NATS_URL is set)", "stream", review.StreamPractice, "err", err)
		return 1
	}
	if practiceCons != nil {
		defer practiceCons.Close()
		s, serr := svc.StartConsumers(ctx, practiceCons)
		if serr != nil {
			logger.Error("start practice consumer", "err", serr)
			return 1
		}
		subs = append(subs, s)
	}

	// Durable pull consumer on XLEARN_REVIEW (xlearn.review.revision_due) — the in-app
	// notifications worker (S07), a separate consumer on review's own stream. It reads
	// back the sweep's revision_due events and writes reminder rows (ADR-0016).
	reviewCons, err := newConsumer(ctx, cfg.NATS.URL, review.StreamReview, logger)
	if err != nil {
		logger.Error("nats consumer init failed; refusing to start (NATS_URL is set)", "stream", review.StreamReview, "err", err)
		return 1
	}
	if reviewCons != nil {
		defer reviewCons.Close()
		s, serr := svc.StartNotifications(ctx, reviewCons)
		if serr != nil {
			logger.Error("start notifications consumer", "err", serr)
			return 1
		}
		subs = append(subs, s)
	}

	// Periodic sweep: materialises due-today reliably even after days offline (R-SR6).
	sweeper := svc.NewSweeper(cfg.Sweep.Interval, cfg.Sweep.Batch)
	go sweeper.Run(ctx)

	// Weekly weak-area recompute: same cadence, but its OWN goroutine so a slow identity
	// or a large account count can't stall the due-sweep above (S07, ADR-0016).
	weakArea := svc.NewWeakAreaWorker(cfg.Sweep.Interval)
	go weakArea.Run(ctx)

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
	logger.Info("review stopped")
	return 0
}

// newPublisher builds the JetStream publisher when NATS_URL is set, else the log
// publisher (local dev / no broker). An unreachable broker is not an error — the
// connection retries and the relay buffers in the outbox — but an init error with
// NATS_URL set (a bad or missing nkey seed, bad options) is returned so the service
// exits: it never falls back to the log publisher, which would mark rows sent.
func newPublisher(ctx context.Context, natsURL string, logger *slog.Logger) (events.Publisher, error) {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; using the log publisher (events are not delivered to JetStream)")
		return events.NewLogPublisher(logger, review.StreamReview), nil
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	return events.NewNatsPublisher(natsCtx, review.ServiceName, natsURL, review.StreamReview, logger)
}

// newConsumer builds a durable JetStream consumer on stream when NATS_URL is set, else
// nil (local dev / no broker: the service runs without consuming). The consumer
// auto-reconnects, so an unreachable broker is not an error (Subscribe retries the
// stream); an init error with NATS_URL set is returned so the service exits (fail
// closed) rather than silently running without its consumers.
func newConsumer(ctx context.Context, natsURL, stream string, logger *slog.Logger) (*events.NatsConsumer, error) {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; consumers are disabled", "stream", stream)
		return nil, nil
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	return events.NewNatsConsumer(natsCtx, review.ServiceName, natsURL, stream, logger)
}

// newPool opens a pgxpool from poolConfig.
func newPool(ctx context.Context, db review.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := poolConfig(db)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// poolConfig pins search_path to the service's schema (defence in depth; all SQL is
// schema-qualified anyway) and MaxConns to PG_MAX_CONNS (default 4, L21).
func poolConfig(db review.DBConfig) (*pgxpool.Config, error) {
	poolCfg, err := pgxpool.ParseConfig(db.DSN())
	if err != nil {
		return nil, err
	}
	if db.SearchPath != "" {
		poolCfg.ConnConfig.RuntimeParams["search_path"] = db.SearchPath
	}
	poolCfg.MaxConns = config.PGMaxConns(config.DefaultPGMaxConns)
	return poolCfg, nil
}

// version is stamped at build time with -ldflags "-X main.version=vX.Y.Z" (see
// deploy/review.Dockerfile). It must be the main-package path: -X on a symbol that
// does not exist is a silent no-op.
var version = "dev"

func buildVersion() string { return version }
