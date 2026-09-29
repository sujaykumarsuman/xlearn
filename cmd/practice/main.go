// Command practice is the xLearn practice service: the per-user problem lifecycle —
// stage gating, server-authoritative timers, the reveal-penalty rule, and outcome
// logging — and the first event producer, emitting problem_solved / attempt_logged
// / solution_revealed_early to NATS JetStream via a transactional outbox
// (ADR-0004/0005/0006). It is a ClusterIP-only internal service (route.enabled:
// false); the gateway is the only caller. Schema `practice` is migrated on startup
// inside a Postgres advisory lock; the service refuses to serve if migration fails.
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

	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/config"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
	"github.com/sujaykumarsuman/xlearn/internal/practice"
	"github.com/sujaykumarsuman/xlearn/internal/practice/store"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
	natsTimeout     = 10 * time.Second
)

func main() {
	// `practice -version` prints the -ldflags-stamped version and exits (guarded by
	// deploy/version_test.go).
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Println(buildVersion())
		return
	}
	os.Exit(run())
}

func run() int {
	cfg := practice.LoadConfig()
	logger := slogx.New(cfg.LogLevel)

	// The compiled-in course manifests resolve the internal `?path=` param (m1-03); load
	// and validate them first so a bad manifest fails the boot, not a request.
	courses, err := course.LoadEmbedded()
	if err != nil {
		logger.Error("course manifests invalid; refusing to start", "err", err)
		return 1
	}

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
	svc := practice.NewService(st, verifier, logger).WithCourses(courses)

	handler := httpx.Chain(svc.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Outbox relay: drains practice events to JetStream (XLEARN_PRACTICE). With no
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

	errCh := make(chan error, 1)
	go func() {
		logger.Info("practice listening", "addr", cfg.Addr(), "version", buildVersion())
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if serr := srv.Shutdown(shutdownCtx); serr != nil {
		logger.Error("graceful shutdown failed", "err", serr)
		return 1
	}
	logger.Info("practice stopped")
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
		return events.NewLogPublisher(logger, practice.StreamPractice), nil
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	return events.NewNatsPublisher(natsCtx, practice.ServiceName, natsURL, practice.StreamPractice, logger)
}

// newPool opens a pgxpool from poolConfig.
func newPool(ctx context.Context, db practice.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := poolConfig(db)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// poolConfig pins search_path to the service's schema (defence in depth; all SQL is
// schema-qualified anyway) and MaxConns to PG_MAX_CONNS (default 4, L21).
func poolConfig(db practice.DBConfig) (*pgxpool.Config, error) {
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
// deploy/practice.Dockerfile). It must be the main-package path: -X on a symbol that
// does not exist is a silent no-op.
var version = "dev"

func buildVersion() string { return version }
