// Command identity is the xLearn identity service: OAuth 2.0/OIDC sign-in with
// GitHub & Google, opaque server-side sessions, accounts + onboarding state, and
// the account_created transactional outbox, relayed to JetStream XLEARN_IDENTITY
// when NATS_URL is set (ADR-0006, ADR-0004/0005, ADR-0035). It is a
// ClusterIP-only internal service (route.enabled: false); the gateway is the only
// caller. Schema `identity` is migrated on startup inside a Postgres advisory lock;
// the service refuses to serve if migration fails.
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
	"github.com/sujaykumarsuman/xlearn/internal/identity"
	"github.com/sujaykumarsuman/xlearn/internal/identity/admin"
	"github.com/sujaykumarsuman/xlearn/internal/identity/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/config"
	"github.com/sujaykumarsuman/xlearn/internal/platform/events"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
	natsTimeout     = 10 * time.Second
)

func main() {
	// `identity -version` prints the -ldflags-stamped version and exits (guarded by
	// deploy/version_test.go).
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Println(buildVersion())
		return
	}
	// `identity admin …` is the owner's CLI (m1-04, ADR-0033 §8), run via kubectl exec in
	// the running pod: the pod's own DB credentials, no migrations, no server.
	if len(os.Args) >= 2 && os.Args[1] == "admin" {
		os.Exit(admin.Main(os.Args[2:], os.Stdout, os.Stderr))
	}
	os.Exit(run())
}

func run() int {
	cfg := identity.LoadConfig()
	logger := slogx.New(cfg.LogLevel)
	// L7 (ADR-0033 §3): `open` without DEV_AUTH runs closed and logs at ERROR (D34: a log
	// line, no alert).
	cfg.Auth.LogSignupMode(logger)

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
	// The compiled-in course manifests enrollment validates against (m1-03). A bad
	// manifest fails the boot, not a request.
	courses, err := course.LoadEmbedded()
	if err != nil {
		logger.Error("load course manifests; refusing to serve", "err", err)
		return 1
	}
	svc := identity.NewService(cfg, st, verifier, courses, logger)

	handler := httpx.Chain(svc.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Outbox relay: drains account_created events to JetStream (XLEARN_IDENTITY) when
	// NATS_URL is set; with no NATS_URL (local dev, and prod until mi-06's identity N2
	// PR) it keeps the log publisher so the outbox drains. With NATS_URL set an init
	// error is fatal (fail closed): the log publisher would mark rows sent.
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
		logger.Info("identity listening", "addr", cfg.Addr(), "version", buildVersion(), "signup_mode", cfg.Auth.Signup)
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
	logger.Info("identity stopped")
	return 0
}

// newPublisher builds the JetStream publisher on XLEARN_IDENTITY when NATS_URL is
// set, else the log publisher — exactly as practice's newPublisher. An unreachable
// broker is not an error (the connection retries; the outbox buffers), but an init
// error with NATS_URL set (a bad or missing nkey seed, bad options) is returned so the
// service exits instead of marking rows sent without delivering them.
func newPublisher(ctx context.Context, natsURL string, logger *slog.Logger) (events.Publisher, error) {
	if natsURL == "" {
		logger.Warn("NATS_URL not set; using the log publisher (events are not delivered to JetStream)")
		return events.NewLogPublisher(logger, identity.StreamIdentity), nil
	}
	natsCtx, cancel := context.WithTimeout(ctx, natsTimeout)
	defer cancel()
	return events.NewNatsPublisher(natsCtx, identity.ServiceName, natsURL, identity.StreamIdentity, logger)
}

// newPool opens a pgxpool from poolConfig.
func newPool(ctx context.Context, db identity.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := poolConfig(db)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// poolConfig pins search_path to the service's schema (defence in depth; all SQL is
// schema-qualified anyway) and MaxConns to PG_MAX_CONNS (default 4, L21).
func poolConfig(db identity.DBConfig) (*pgxpool.Config, error) {
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
// deploy/identity.Dockerfile). It must be the main-package path: -X on a symbol that
// does not exist is a silent no-op.
var version = "dev"

func buildVersion() string { return version }
