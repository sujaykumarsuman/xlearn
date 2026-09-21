// Package httpx holds the shared HTTP server plumbing: a hardened http.Server
// builder and a small middleware chain (request-id, panic-recovery, access log).
// Every xLearn service wraps its handler with Chain so logging and resilience
// behave identically across the fleet.
package httpx

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/platform/slogx"
)

// RequestIDHeader is the response header the request-id is echoed on.
const RequestIDHeader = "X-Request-Id"

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// NewServer returns an http.Server with sane production timeouts. The caller
// owns ListenAndServe and Shutdown.
func NewServer(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

// Chain applies middlewares so that mws[0] is the outermost wrapper (runs first
// on the way in, last on the way out).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// RequestID assigns each request a request-id (reusing an inbound
// X-Request-Id when present), stores it in the context for the logger, and
// echoes it on the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			id = newID()
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(slogx.WithRequestID(r.Context(), id)))
	})
}

// Recoverer converts a panic in a downstream handler into a 500 and logs it
// with the request-id, so one bad handler can't take the process down.
func Recoverer(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						slogx.RequestIDField, slogx.RequestID(r.Context()),
						"method", r.Method, "path", r.URL.Path, "panic", rec)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					_, _ = w.Write([]byte(`{"error":{"code":"internal"}}`))
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// AccessLog logs one line per request (method, path, status, duration,
// request-id). Liveness/readiness probes are skipped to keep logs signal-rich.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("request",
				slogx.RequestIDField, slogx.RequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"bytes", sw.bytes,
				"duration_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}

// statusWriter captures the response status code and byte count for logging.
type statusWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (s *statusWriter) WriteHeader(code int) {
	if !s.wroteHeader {
		s.status = code
		s.wroteHeader = true
	}
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Write(b []byte) (int, error) {
	if !s.wroteHeader {
		s.wroteHeader = true
	}
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

// Unwrap exposes the wrapped ResponseWriter so http.ResponseController can reach the
// base writer's Flusher / deadline setter through this middleware. Without it, a
// streaming handler (the coach SSE endpoint, and the gateway's SSE proxy) could not
// flush or clear the server WriteTimeout and the stream would buffer or stall.
func (s *statusWriter) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// newID returns a random 16-byte hex request-id.
func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should not fail; fall back to a fixed marker so the field
		// is never empty rather than aborting the request.
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}
