package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// Request-body caps (ADR-0035 §4 L6). Every client body the gateway reads goes through
// ReadBody (or, for a streaming proxy, http.MaxBytesReader with one of these), so an
// oversize body gets a typed 413 instead of being silently truncated. hack/lint-bodies.sh
// keeps raw r.Body reads out of internal/gateway and internal/platform.
//
// The registry grows by milestone: M3 adds the judge scene (512 KiB) and export (32 KiB)
// caps and m3-09 the 64 KiB code cap.
const (
	// BodyLimitDefault is the cap for every gateway request body without its own entry.
	BodyLimitDefault int64 = 1 << 20
	// BodyLimitCanvas caps a system-design canvas save.
	BodyLimitCanvas int64 = 640 << 10
	// BodyLimitInterviewSnapshot caps an interviewer snapshot.
	BodyLimitInterviewSnapshot int64 = 64 << 10
)

// CodeBodyTooLarge is the 413 error code.
const CodeBodyTooLarge = "body_too_large"

// ReadBody reads r's body up to limit bytes. On success it returns the bytes and true. An
// oversize body (a declared Content-Length over limit, or more than limit bytes read) gets
// the typed 413 {"error":{"code":"body_too_large","limit":N}}; any other read error gets
// 400 bad_request. In both cases ReadBody has written the response and returns false.
func ReadBody(w http.ResponseWriter, r *http.Request, limit int64) ([]byte, bool) {
	if r.ContentLength > limit {
		WriteBodyTooLarge(w, limit)
		return nil, false
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		if IsBodyTooLarge(err) {
			WriteBodyTooLarge(w, limit)
			return nil, false
		}
		writeJSONError(w, http.StatusBadRequest, map[string]any{"code": "bad_request", "message": "could not read body"})
		return nil, false
	}
	return b, true
}

// IsBodyTooLarge reports whether err is (or wraps) the error http.MaxBytesReader returns
// past its limit — including when an http.Client surfaces it from a proxied request body.
func IsBodyTooLarge(err error) bool {
	var mbe *http.MaxBytesError
	return errors.As(err, &mbe)
}

// WriteBodyTooLarge writes the typed 413.
func WriteBodyTooLarge(w http.ResponseWriter, limit int64) {
	writeJSONError(w, http.StatusRequestEntityTooLarge, map[string]any{
		"code":    CodeBodyTooLarge,
		"message": "request body too large",
		"limit":   limit,
	})
}

func writeJSONError(w http.ResponseWriter, status int, body map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": body})
}
