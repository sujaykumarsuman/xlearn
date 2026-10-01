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
	"fmt"
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
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/config"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

const (
	shutdownTimeout = 10 * time.Second
	migrateTimeout  = 60 * time.Second
)

func main() {
	// `coach -version` prints the -ldflags-stamped version and exits (guarded by
	// deploy/version_test.go).
	if len(os.Args) == 2 && os.Args[1] == "-version" {
		fmt.Println(buildVersion())
		return
	}
	os.Exit(run())
}

func run() int {
	cfg := coach.LoadConfig()
	logger := slogx.New(cfg.LogLevel)

	// The envelope master key is required (ADR-0007). Refuse to serve without a valid
	// 32-byte key — the service cannot seal/open provider keys otherwise, and running
	// with an ephemeral key would silently make every stored key unrecoverable.
	cipher, keyring, err := loadKeys(cfg, logger)
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

	// The compiled-in course manifests (thread keys and path_slug, m1-03). A bad manifest
	// fails the boot, not a request.
	courses, err := course.LoadEmbedded()
	if err != nil {
		logger.Error("load course manifests; refusing to serve", "err", err)
		return 1
	}

	svc := coach.NewService(st, verifier, cipher, keyring, openai, anthropic, courses, logger)

	handler := httpx.Chain(svc.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The background AD re-wrap (m1-10): backfills the AD-bound sealed pair migration
	// 00006 added, and afterwards repairs rows left stale by a rollback key-replace or a
	// master-key rotation. A GOROUTINE in this pod, not a new workload — no new container
	// and no change to the memory sum (ADR-0035 §5). It ends with ctx on shutdown; a pass
	// in flight holds a transaction-scoped advisory lock that COMMIT/ROLLBACK releases, so
	// a rolling update never leaks it.
	go coach.NewRewrapper(st, cipher, keyring, logger).Run(ctx)

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

// loadKeys resolves BOTH halves of coach's key material (ADR-0007, ADR-0031 §7):
//
//   - the LEGACY cipher from COACH_MASTER_KEY / COACH_MASTER_KEY_FILE. Still REQUIRED,
//     and not merely for backwards compatibility: it is the only thing that can open the
//     unbound pair, which is what every key the re-wrap job has not reached yet is stored
//     under, and what keeps v1.6.0 a valid rollback target (ADR-0034 §3). It is dropped
//     when l-02 drops the legacy pair, not before.
//
//   - the KEK KEYRING from COACH_MASTER_KEYS / COACH_MASTER_KEYS_FILE
//     ("k1:<base64>,k0:<base64>", first entry active). When unset — which is how
//     PRODUCTION runs today, since this sprint changes no SOPS secret — the keyring is
//     {k0: COACH_MASTER_KEY} with k0 active, so today's key wraps the AD pairs and a
//     rotation later is a secret change rather than a code change.
//
// Neither key is ever logged; only the keyring's entry IDS, which carry no material.
func loadKeys(cfg coach.Config, logger *slog.Logger) (*secrets.Cipher, *secrets.Keyring, error) {
	raw, err := readKeyMaterial(cfg.MasterKeyRaw, coach.MasterKeyFile())
	if err != nil {
		return nil, nil, err
	}
	key, err := secrets.ParseMasterKey(raw)
	if err != nil {
		return nil, nil, err
	}
	defer secrets.Zero(key)
	cipher, err := secrets.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("envelope master key loaded")

	spec, err := readKeyMaterial(coach.MasterKeysSpec(), coach.MasterKeysFile())
	if err != nil {
		return nil, nil, err
	}
	if spec == "" {
		keyring := secrets.KeyringOf(coach.DefaultKEKID, cipher)
		logger.Info("coach keyring: single entry from COACH_MASTER_KEY", "active_kek", keyring.Active())
		return cipher, keyring, nil
	}
	keyring, err := secrets.ParseKeyring(spec)
	if err != nil {
		return nil, nil, err
	}
	logger.Info("coach keyring loaded", "active_kek", keyring.Active(), "entries", keyring.IDs())
	return cipher, keyring, nil
}

// readKeyMaterial prefers an env value and falls back to a mounted file. It returns ""
// when neither is set, so the caller decides whether that is fatal.
func readKeyMaterial(env, file string) (string, error) {
	if v := strings.TrimSpace(env); v != "" {
		return v, nil
	}
	if file == "" {
		return "", nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// newPool opens a pgxpool and pins search_path to the service's schema (defence in
// depth; all SQL is schema-qualified anyway).
func newPool(ctx context.Context, db coach.DBConfig) (*pgxpool.Pool, error) {
	poolCfg, err := poolConfig(db)
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(ctx, poolCfg)
}

// poolConfig pins search_path to the service's schema and MaxConns to PG_MAX_CONNS
// (default 4, L21).
func poolConfig(db coach.DBConfig) (*pgxpool.Config, error) {
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
// deploy/coach.Dockerfile). It must be the main-package path: -X on a symbol that
// does not exist is a silent no-op.
var version = "dev"

func buildVersion() string { return version }
