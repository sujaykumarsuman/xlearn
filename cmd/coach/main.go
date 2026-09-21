// Command coach is the xLearn AI-coach service (ADR-0007). It stores each user's own
// provider API key ENVELOPE-ENCRYPTED at rest (a per-record data key seals the key with
// XChaCha20-Poly1305; the data key is wrapped by a SOPS-managed master key mounted to
// the pod), builds page-context prompts server-side (Socratic + spoiler-free during an
// active attempt, code reviewer post-solve — the mode is set authoritatively by the
// gateway, never the client), and streams the reply from the user's LLM provider
// (OpenAI / Anthropic) back over SSE. It never returns or logs the raw key. ClusterIP-
// only internal service (route.enabled: false); the gateway is the only HTTP caller.
// Schema `coach` is migrated on startup inside a Postgres advisory lock; the service
// refuses to serve if migration fails or the master key is missing/invalid. coach emits
// no events (no outbox) and consumes none via NATS.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/sujaykumarsuman/xlearn/internal/coach"
	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
)

func main() { os.Exit(run()) }

func run() int {
	cfg := coach.LoadConfig()
	logger := slogx.New(cfg.LogLevel)

	// The envelope master key is required (ADR-0007). Refuse to serve without a valid
	// 32-byte key — the service cannot seal/open provider keys otherwise, and running
	// with an ephemeral key would silently make every stored key unrecoverable.
	cipher, err := loadCipher(cfg, logger)
	if err != nil {
		logger.Error("load master key; refusing to serve", "err", err)
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

	// Streaming provider clients: NO overall client timeout (SSE); per-call bounds come
	// from the request context. A shared Transport keeps connections pooled.
	providerHTTP := &http.Client{Transport: http.DefaultTransport.(*http.Transport).Clone()}
	openai := coach.NewOpenAIProvider(cfg.Providers.OpenAIBaseURL, providerHTTP)
	anthropic := coach.NewAnthropicProvider(cfg.Providers.AnthropicBaseURL, providerHTTP)

	svc := coach.NewService(st, verifier, cipher, openai, anthropic, logger)

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
		logger.Info("coach listening", "addr", cfg.Addr(), "version", buildVersion())
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
	logger.Info("coach stopped")
	return 0
}

// loadCipher resolves the envelope master key (ADR-0007): from COACH_MASTER_KEY (env),
// then COACH_MASTER_KEY_FILE (a mounted secret file), decodes it (base64/hex/raw) to
// exactly 32 bytes, and builds the Cipher. The key material is never logged.
func loadCipher(cfg coach.Config, logger *slog.Logger) (*secrets.Cipher, error) {
	raw := strings.TrimSpace(cfg.MasterKeyRaw)
	if raw == "" {
		if f := coach.MasterKeyFile(); f != "" {
			b, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			raw = strings.TrimSpace(string(b))
		}
	}
	key, err := secrets.ParseMasterKey(raw)
	if err != nil {
		return nil, err
	}
	defer secrets.Zero(key)
	logger.Info("envelope master key loaded")
	return secrets.NewCipher(key)
}

// newPool opens a pgxpool and pins search_path to the service's schema (defence in
// depth; all SQL is schema-qualified anyway).
func newPool(ctx context.Context, db coach.DBConfig) (*pgxpool.Pool, error) {
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
