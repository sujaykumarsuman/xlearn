// Command gateway is the xLearn edge service: it serves the embedded React SPA
// and the BFF API under the base path (default /xlearn) and exposes the k8s
// liveness/readiness probes. It is the only internet-facing xLearn service
// (ADR-0008/0009); later sprints add session auth, JWT minting and screen
// aggregation. This binary is stateless — it serves entirely from the embedded
// filesystem and writes nothing to disk (read-only rootfs, ADR-0009).
package main

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sujaykumarsuman/xlearn"
	"github.com/sujaykumarsuman/xlearn/internal/gateway"
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

	gw := gateway.New(gateway.Options{
		BasePath: cfg.BasePath,
		Version:  xlearn.Version,
		Dist:     dist,
		Logger:   logger,
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
