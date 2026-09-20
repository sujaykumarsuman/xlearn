// Command curriculum is the xLearn curriculum service: the read-heavy content rail
// — paths, phases, weeks, concepts, problems and their stage-scoped sections
// (ADR-0005). It is a ClusterIP-only internal service (route.enabled: false); the
// gateway is the only caller. Schema `curriculum` is migrated on startup inside a
// Postgres advisory lock, then the versioned seed (curriculum/) is applied
// idempotently; the service refuses to serve if either fails. Curriculum emits no
// events (no outbox).
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	seeddata "github.com/sujaykumarsuman/xlearn/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum"
	"github.com/sujaykumarsuman/xlearn/internal/curriculum/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
	seedTimeout     = 60 * time.Second
)

func main() { os.Exit(run()) }

func run() int {
	cfg := curriculum.LoadConfig()
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

	// Seed the versioned curriculum idempotently; a content service with no content
	// is broken, so refuse to serve on failure.
	seedCtx, cancelSeed := context.WithTimeout(context.Background(), seedTimeout)
	defer cancelSeed()
	if err := curriculum.Seed(seedCtx, st, seeddata.FS, logger); err != nil {
		logger.Error("seed failed; refusing to serve", "err", err)
		return 1
	}

	svc := curriculum.NewService(st, logger)

	handler := httpx.Chain(svc.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("curriculum listening", "addr", cfg.Addr(), "version", buildVersion())
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
	logger.Info("curriculum stopped")
	return 0
}

// newPool opens a pgxpool and pins search_path to the service's schema (defence in
// depth; all SQL is schema-qualified anyway).
func newPool(ctx context.Context, db curriculum.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(db.DSN())
	if err != nil {
		return nil, err
	}
	if db.SearchPath != "" {
		poolCfg.ConnConfig.RuntimeParams["search_path"] = db.SearchPath
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// buildVersion is stamped via -ldflags at build time (mirrors the gateway/identity).
var version = "dev"

func buildVersion() string { return version }
