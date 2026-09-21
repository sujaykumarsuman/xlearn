package coach

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// openAISSE writes an OpenAI-style chat.completions stream that says "Hi there".
func openAISSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fl, _ := w.(http.Flusher)
	for _, chunk := range []string{
		`{"choices":[{"delta":{"content":"Hi"}}]}`,
		`{"choices":[{"delta":{"content":" there"}}]}`,
		`[DONE]`,
	} {
		_, _ = w.Write([]byte("data: " + chunk + "\n\n"))
		if fl != nil {
			fl.Flush()
		}
	}
}

// anthropicSSE writes an Anthropic-style messages stream that says "Hi there".
func anthropicSSE(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fl, _ := w.(http.Flusher)
	frames := []string{
		`event: message_start` + "\n" + `data: {"type":"message_start"}`,
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"Hi"}}`,
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":" there"}}`,
		`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
	}
	for _, f := range frames {
		_, _ = w.Write([]byte(f + "\n\n"))
		if fl != nil {
			fl.Flush()
		}
	}
}

func collect(t *testing.T, p Provider, apiKey string) (string, error) {
	t.Helper()
	var got strings.Builder
	err := p.Stream(context.Background(), apiKey, ChatRequest{Model: "m", System: "sys", Messages: []ChatMessage{{Role: "user", Content: "hello"}}}, func(d string) error {
		got.WriteString(d)
		return nil
	})
	return got.String(), err
}

func TestOpenAIProviderStreams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		openAISSE(w)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, srv.Client())
	got, err := collect(t, p, "sk-good")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got != "Hi there" {
		t.Fatalf("accumulated = %q, want %q", got, "Hi there")
	}
}

func TestAnthropicProviderStreams(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "sk-good" || r.Header.Get("anthropic-version") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		anthropicSSE(w)
	}))
	defer srv.Close()

	p := NewAnthropicProvider(srv.URL, srv.Client())
	got, err := collect(t, p, "sk-good")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got != "Hi there" {
		t.Fatalf("accumulated = %q, want %q", got, "Hi there")
	}
}

func TestProviderAuthFailureMapsToErrProviderAuth(t *testing.T) {
	for _, tc := range []struct {
		name string
		make func(url string, c *http.Client) Provider
	}{
		{"openai", func(u string, c *http.Client) Provider { return NewOpenAIProvider(u, c) }},
		{"anthropic", func(u string, c *http.Client) Provider { return NewAnthropicProvider(u, c) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
			}))
			defer srv.Close()
			p := tc.make(srv.URL, srv.Client())
			_, err := collect(t, p, "sk-bad")
			if !errors.Is(err, ErrProviderAuth) {
				t.Fatalf("err = %v, want ErrProviderAuth", err)
			}
		})
	}
}

func TestOpenAIInlineErrorEvent(t *testing.T) {
	// An auth error frame mid-stream must surface as ErrProviderAuth (not a silent
	// truncated success).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"error":{"message":"bad key","type":"authentication_error"}}` + "\n\n"))
	}))
	defer srv.Close()
	p := NewOpenAIProvider(srv.URL, srv.Client())
	if _, err := collect(t, p, "sk-bad"); !errors.Is(err, ErrProviderAuth) {
		t.Fatalf("openai inline auth error = %v, want ErrProviderAuth", err)
	}

	// A non-auth inline error must surface as a generic error, not a success.
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"error":{"message":"boom","type":"server_error"}}` + "\n\n"))
	}))
	defer srv2.Close()
	p2 := NewOpenAIProvider(srv2.URL, srv2.Client())
	if _, err := collect(t, p2, "sk-ok"); err == nil || errors.Is(err, ErrProviderAuth) {
		t.Fatalf("openai inline server error = %v, want a non-nil non-auth error", err)
	}
}

func TestAnthropicInlineErrorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`event: error` + "\n" + `data: {"type":"error","error":{"type":"authentication_error","message":"bad key"}}` + "\n\n"))
	}))
	defer srv.Close()
	p := NewAnthropicProvider(srv.URL, srv.Client())
	_, err := collect(t, p, "sk-bad")
	if !errors.Is(err, ErrProviderAuth) {
		t.Fatalf("inline error err = %v, want ErrProviderAuth", err)
	}
}
