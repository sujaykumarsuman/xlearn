// Package slogx builds the shared structured logger: a JSON slog handler writing
// to stdout, and helpers to carry a request-id through the context so every log
// line for a request can be correlated (the logging concern in ADR-0009). The
// request-id is propagated gateway->services on internal calls in later sprints.
package slogx

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// ctxKey is the private context key for the request-id.
type ctxKey struct{}

// RequestIDField is the slog attribute key used for the request-id.
const RequestIDField = "request_id"

// New returns a JSON slog.Logger writing to stdout at the given level
// (debug|info|warn|error; unknown values fall back to info).
func New(level string) *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parseLevel(level)})
	return slog.New(h)
}

// parseLevel maps a string level to slog.Level, defaulting to Info.
func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a context carrying the request-id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// RequestID returns the request-id carried by ctx, or "" if none.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}
