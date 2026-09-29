package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type errEnvelope struct {
	Error struct {
		Code  string `json:"code"`
		Limit int64  `json:"limit"`
	} `json:"error"`
}

func TestReadBodyWithinLimit(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"a":1}`))
	w := httptest.NewRecorder()
	b, ok := ReadBody(w, r, 16)
	if !ok || string(b) != `{"a":1}` {
		t.Fatalf("ReadBody = %q, %v", b, ok)
	}
	// Exactly at the limit is fine.
	r = httptest.NewRequest("POST", "/", strings.NewReader(strings.Repeat("x", 16)))
	if b, ok := ReadBody(httptest.NewRecorder(), r, 16); !ok || len(b) != 16 {
		t.Fatalf("at-limit body refused: %d, %v", len(b), ok)
	}
}

func TestReadBodyOversizeIsTyped413(t *testing.T) {
	for _, tc := range []struct {
		name    string
		chunked bool
	}{{"declared content-length", false}, {"chunked overrun", true}} {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader = strings.NewReader(strings.Repeat("x", 17))
			if tc.chunked {
				body = io.MultiReader(body) // hides the length: ContentLength stays unknown
			}
			r := httptest.NewRequest("POST", "/", body)
			if tc.chunked {
				r.ContentLength = -1
			}
			w := httptest.NewRecorder()
			if _, ok := ReadBody(w, r, 16); ok {
				t.Fatal("oversize body accepted")
			}
			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("status %d, want 413", w.Code)
			}
			var env errEnvelope
			if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
				t.Fatal(err)
			}
			if env.Error.Code != CodeBodyTooLarge || env.Error.Limit != 16 {
				t.Fatalf("envelope %+v", env.Error)
			}
		})
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("connection reset") }

func TestReadBodyOtherErrorIs400(t *testing.T) {
	r := httptest.NewRequest("POST", "/", failingReader{})
	r.ContentLength = -1
	w := httptest.NewRecorder()
	if _, ok := ReadBody(w, r, 16); ok {
		t.Fatal("a failing body was accepted")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", w.Code)
	}
}

func TestIsBodyTooLargeUnwraps(t *testing.T) {
	err := fmt.Errorf("proxy: %w", &http.MaxBytesError{Limit: 1})
	if !IsBodyTooLarge(err) {
		t.Fatal("wrapped MaxBytesError not detected")
	}
	if IsBodyTooLarge(errors.New("other")) {
		t.Fatal("unrelated error detected")
	}
}

func TestBodyLimitRegistry(t *testing.T) {
	if BodyLimitDefault != 1<<20 || BodyLimitCanvas != 640<<10 || BodyLimitInterviewSnapshot != 64<<10 {
		t.Fatal("the L6 registry drifted from ADR-0035 §4")
	}
}
