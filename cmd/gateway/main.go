// Command gateway is the xLearn edge service: it serves the embedded React SPA
// and the BFF API under the base path (default /xlearn) and exposes the k8s
// liveness/readiness probes. It is the only internet-facing xLearn service
// (ADR-0008/0009); later sprints add session auth, JWT minting and screen
// aggregation. This binary is stateless — it serves entirely from the embedded
// filesystem and writes nothing to disk (read-only rootfs, ADR-0009).
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sujaykumarsuman/xlearn"
	"github.com/sujaykumarsuman/xlearn/internal/gateway"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/config"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

// shutdownTimeout bounds graceful drain on SIGINT/SIGTERM.
const shutdownTimeout = 10 * time.Second

func main() {
	os.Exit(run())
}

func run() int {
	cfg := config.Load()
	logger := slogx.New(cfg.LogLevel)

	dist, err := fs.Sub(xlearn.Dist, "web/dist")
	if err != nil {
		logger.Error("embed web/dist", "err", err)
		return 1
	}

	signer, err := loadSigner(cfg.JWT, logger)
	if err != nil {
		logger.Error("load jwt signing key", "err", err)
		return 1
	}

	gw := gateway.New(gateway.Options{
		BasePath:           cfg.BasePath,
		Version:            xlearn.Version,
		Dist:               dist,
		Logger:             logger,
		Signer:             signer,
		IdentityBaseURL:    cfg.IdentityBaseURL,
		AudienceIdentity:   cfg.JWT.AudienceIdentity,
		CurriculumBaseURL:  cfg.CurriculumBaseURL,
		PracticeBaseURL:    cfg.PracticeBaseURL,
		AudiencePractice:   cfg.JWT.AudiencePractice,
		ReviewBaseURL:      cfg.ReviewBaseURL,
		AudienceReview:     cfg.JWT.AudienceReview,
		AssessmentBaseURL:  cfg.AssessmentBaseURL,
		AudienceAssessment: cfg.JWT.AudienceAssessment,
		CoachBaseURL:       cfg.CoachBaseURL,
		AudienceCoach:      cfg.JWT.AudienceCoach,
		AggCacheTTL:        cfg.AggCacheTTL,
	})

	handler := httpx.Chain(gw.Handler(),
		httpx.RequestID,
		httpx.AccessLog(logger),
		httpx.Recoverer(logger),
	)
	srv := httpx.NewServer(cfg.Addr(), handler)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("gateway listening",
			"addr", cfg.Addr(), "base_path", cfg.BasePath, "version", xlearn.Version)
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
	logger.Info("gateway stopped")
	return 0
}

// loadSigner resolves the RSA signing key (ADR-0006): from JWT_PRIVATE_KEY (PEM),
// then JWT_PRIVATE_KEY_FILE, else an ephemeral dev key (logged loudly — tokens do
// not survive a restart and other pods can't verify them, so prod must set one).
func loadSigner(cfg config.JWTConfig, logger *slog.Logger) (*auth.Signer, error) {
	var pemBytes []byte
	switch {
	case cfg.PrivateKeyPEM != "":
		pemBytes = []byte(cfg.PrivateKeyPEM)
	case cfg.PrivateKeyFile != "":
		b, err := os.ReadFile(cfg.PrivateKeyFile)
		if err != nil {
			return nil, err
		}
		pemBytes = b
	default:
		logger.Warn("no JWT signing key configured; generating an ephemeral dev key (set JWT_PRIVATE_KEY in prod)")
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		return auth.NewSigner(key, cfg.Issuer, cfg.TTL), nil
	}
	key, err := auth.ParseRSAPrivateKeyPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	signer := auth.NewSigner(key, cfg.Issuer, cfg.TTL)
	// Publish any additional verification keys (rotation overlap window, ADR-0011).
	if cfg.AdditionalPublicKeysPEM != "" {
		pubs, err := auth.ParseRSAPublicKeysPEM([]byte(cfg.AdditionalPublicKeysPEM))
		if err != nil {
			return nil, err
		}
		signer.AddVerificationKeys(pubs...)
		logger.Info("publishing additional JWKS verification keys", "count", len(pubs))
	}
	return signer, nil
}
