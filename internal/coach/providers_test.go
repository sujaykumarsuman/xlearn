package coach

import (
	"context"
	"encoding/json"
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
		`{"choices":[{"delta":{},"finish_reason":"stop"}]}`,
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
	anthropicFrames(w, "end_turn", true)
}

// anthropicFrames writes an Anthropic messages stream: a (display-omitted) thinking
// block, the text "Hi there", a message_delta carrying stopReason, and — when stop is
// true — the terminal message_stop.
func anthropicFrames(w http.ResponseWriter, stopReason string, stop bool) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fl, _ := w.(http.Flusher)
	frames := []string{
		`event: message_start` + "\n" + `data: {"type":"message_start","message":{"stop_reason":null}}`,
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":""}}`,
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"signature_delta","signature":"EqQB"}}`,
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"Hi"}}`,
		`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":" there"}}`,
		`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"` + stopReason + `","stop_sequence":null},"usage":{"output_tokens":15}}`,
	}
	if stop {
		frames = append(frames, `event: message_stop`+"\n"+`data: {"type":"message_stop"}`)
	}
	for _, f := range frames {
		_, _ = w.Write([]byte(f + "\n\n"))
		if fl != nil {
			fl.Flush()
		}
	}
}

func collect(t *testing.T, p Provider, apiKey string) (string, StreamResult, error) {
	t.Helper()
	return collectModel(t, p, apiKey, "m")
}

func collectModel(t *testing.T, p Provider, apiKey, model string) (string, StreamResult, error) {
	t.Helper()
	var got strings.Builder
	res, err := p.Stream(context.Background(), apiKey, ChatRequest{Model: model, System: "sys", Messages: []ChatMessage{{Role: "user", Content: "hello"}}}, func(d string) error {
		got.WriteString(d)
		return nil
	})
	return got.String(), res, err
}

// providers builds each provider against url (for tests that run the same case on both).
var providers = []struct {
	name string
	make func(url string, c *http.Client) Provider
}{
	{"openai", func(u string, c *http.Client) Provider { return NewOpenAIProvider(u, c) }},
	{"anthropic", func(u string, c *http.Client) Provider { return NewAnthropicProvider(u, c) }},
}

func TestOpenAIProviderStreams(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-good" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&sent)
		openAISSE(w)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, srv.Client())
	got, res, err := collectModel(t, p, "sk-good", "gpt-5.6-sol")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if got != "Hi there" {
		t.Fatalf("accumulated = %q, want %q", got, "Hi there")
	}
	if sent["reasoning_effort"] != "low" {
		t.Fatalf("reasoning_effort = %v, want low", sent["reasoning_effort"])
	}
	if res.Truncated || res.StopReason != "stop" {
		t.Fatalf("result = %+v, want stop, not truncated", res)
	}
	// The reply cap is max_completion_tokens (reasoning models reject max_tokens).
	if sent["max_completion_tokens"] != float64(maxProviderTokens) {
		t.Fatalf("max_completion_tokens = %v, want %d", sent["max_completion_tokens"], maxProviderTokens)
	}
	if _, ok := sent["max_tokens"]; ok {
		t.Fatal("openai request must not send the deprecated max_tokens")
	}
}

func TestAnthropicProviderStreams(t *testing.T) {
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "sk-good" || r.Header.Get("anthropic-version") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&sent)
		anthropicSSE(w)
	}))
	defer srv.Close()

	p := NewAnthropicProvider(srv.URL, srv.Client())
	got, res, err := collectModel(t, p, "sk-good", "claude-sonnet-5")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	// Thinking / signature deltas are never forwarded — only the visible text.
	if got != "Hi there" {
		t.Fatalf("accumulated = %q, want %q", got, "Hi there")
	}
	if res.Truncated || res.StopReason != "end_turn" {
		t.Fatalf("result = %+v, want end_turn, not truncated", res)
	}
	if sent["max_tokens"] != float64(maxProviderTokens) {
		t.Fatalf("max_tokens = %v, want %d", sent["max_tokens"], maxProviderTokens)
	}
	oc, _ := sent["output_config"].(map[string]any)
	if oc["effort"] != "low" {
		t.Fatalf("output_config = %v, want effort low", sent["output_config"])
	}
}

func TestAnthropicEffortOnlyForSupportedModels(t *testing.T) {
	for model, want := range map[string]bool{
		"claude-sonnet-5":           true,
		"claude-opus-5":             true,
		"claude-opus-5-5":           true,
		"claude-opus-4-8":           true,
		"claude-opus-4-5-20251101":  true,
		"claude-sonnet-4-6":         true,
		"claude-fable-5-1":          true,
		"claude-haiku-4-5":          false,
		"claude-haiku-4-5-20251001": false,
		"claude-sonnet-4-5":         false,
		"claude-opus-4-1":           false,
		"":                          false,
	} {
		if got := anthropicSupportsEffort(model); got != want {
			t.Errorf("anthropicSupportsEffort(%q) = %v, want %v", model, got, want)
		}
	}

	// A model without effort support gets no output_config (it would 400).
	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&sent)
		anthropicSSE(w)
	}))
	defer srv.Close()
	if _, _, err := collectModel(t, NewAnthropicProvider(srv.URL, srv.Client()), "sk", "claude-haiku-4-5"); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if _, ok := sent["output_config"]; ok {
		t.Fatalf("output_config sent to claude-haiku-4-5: %v", sent["output_config"])
	}
	if sent["max_tokens"] != float64(maxProviderTokens) {
		t.Fatalf("max_tokens = %v, want %d", sent["max_tokens"], maxProviderTokens)
	}
}

func TestOpenAIReasoningEffortOnlyForReasoningModels(t *testing.T) {
	for model, want := range map[string]bool{
		"gpt-5.6-sol":       true,
		"gpt-5.6-luna":      true,
		"gpt-5":             true,
		"gpt-6-astra":       true,
		"gpt-5-chat-latest": false,
		"gpt-4o":            false,
		"gpt-4.1-mini":      false,
		"":                  false,
	} {
		if got := openAISupportsReasoningEffort(model); got != want {
			t.Errorf("openAISupportsReasoningEffort(%q) = %v, want %v", model, got, want)
		}
	}

	var sent map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&sent)
		openAISSE(w)
	}))
	defer srv.Close()
	if _, _, err := collectModel(t, NewOpenAIProvider(srv.URL, srv.Client()), "sk", "gpt-4o"); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if _, ok := sent["reasoning_effort"]; ok {
		t.Fatalf("reasoning_effort sent to gpt-4o: %v", sent["reasoning_effort"])
	}
	if sent["max_completion_tokens"] != float64(maxProviderTokens) {
		t.Fatalf("max_completion_tokens = %v, want %d", sent["max_completion_tokens"], maxProviderTokens)
	}
}

func TestProviderAuthFailureMapsToErrProviderAuth(t *testing.T) {
	for _, tc := range providers {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
			}))
			defer srv.Close()
			p := tc.make(srv.URL, srv.Client())
			_, _, err := collect(t, p, "sk-bad")
			if !errors.Is(err, ErrProviderAuth) {
				t.Fatalf("err = %v, want ErrProviderAuth", err)
			}
		})
	}
}

// TestProviderStatusClassification pins which upstream rejections disable a key (auth)
// and which keep it (limited / unavailable). Only 401, invalid_api_key and
// authentication_error may disable a key.
func TestProviderStatusClassification(t *testing.T) {
	cases := []struct {
		provider string
		name     string
		status   int
		body     string
		want     ProviderErrorKind
	}{
		// OpenAI
		{"openai", "401 invalid_api_key", 401, `{"error":{"message":"Incorrect API key provided: sk-abc***wxyz.","type":"invalid_request_error","param":null,"code":"invalid_api_key"}}`, KindAuth},
		{"openai", "401 no body", 401, ``, KindAuth},
		{"openai", "429 insufficient_quota", 429, `{"error":{"message":"You exceeded your current quota, please check your plan and billing details.","type":"insufficient_quota","param":null,"code":"insufficient_quota"}}`, KindLimited},
		{"openai", "429 credit_balance_exhausted", 429, `{"error":{"message":"You have run out of credits.","type":"insufficient_quota","param":null,"code":"credit_balance_exhausted"}}`, KindLimited},
		{"openai", "429 org spend limit", 429, `{"error":{"message":"Organization spend limit reached","type":"insufficient_quota","param":null,"code":"organization_spend_limit_exceeded"}}`, KindLimited},
		{"openai", "429 project spend limit", 429, `{"error":{"message":"Project spend limit reached","type":"insufficient_quota","param":null,"code":"project_spend_limit_exceeded"}}`, KindLimited},
		{"openai", "403 model_not_found", 403, `{"error":{"message":"Project does not have access to model gpt-5.6-sol","type":"invalid_request_error","param":null,"code":"model_not_found"}}`, KindLimited},
		{"openai", "503 server_is_overloaded", 503, `{"error":{"message":"overloaded","type":"service_unavailable_error","param":null,"code":"server_is_overloaded"}}`, KindUnavailable},
		{"openai", "400 unsupported_parameter", 400, `{"error":{"message":"Unsupported parameter","type":"invalid_request_error","param":"max_tokens","code":"unsupported_parameter"}}`, KindUnavailable},
		{"openai", "429 rate_limit_exceeded", 429, `{"error":{"message":"Rate limit reached","type":"requests","param":null,"code":"rate_limit_exceeded"}}`, KindLimited},
		{"openai", "400 billing_hard_limit_reached", 400, `{"error":{"message":"Billing hard limit has been reached","type":"invalid_request_error","param":null,"code":"billing_hard_limit_reached"}}`, KindLimited},
		{"openai", "402 payment required", 402, `{"error":{"message":"payment required","type":"billing_error","code":null}}`, KindLimited},
		{"openai", "403 unsupported region", 403, `{"error":{"message":"Country, region, or territory not supported","type":"request_forbidden","param":null,"code":"unsupported_country_region_territory"}}`, KindLimited},
		{"openai", "404 model_not_found", 404, `{"error":{"message":"The model does not exist or you do not have access to it.","type":"invalid_request_error","param":null,"code":"model_not_found"}}`, KindLimited},
		{"openai", "400 other", 400, `{"error":{"message":"bad","type":"invalid_request_error","param":"messages","code":null}}`, KindUnavailable},
		{"openai", "500 server_error", 500, `{"error":{"message":"boom","type":"server_error","code":null}}`, KindUnavailable},
		{"openai", "503 no body", 503, ``, KindUnavailable},
		// Anthropic
		{"anthropic", "401 authentication_error", 401, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`, KindAuth},
		{"anthropic", "402 billing_error", 402, `{"type":"error","error":{"type":"billing_error","message":"There's an issue with your billing or payment information."}}`, KindLimited},
		{"anthropic", "403 permission_error", 403, `{"type":"error","error":{"type":"permission_error","message":"Your API key does not have permission to use the specified resource."}}`, KindLimited},
		{"anthropic", "429 rate_limit_error", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"Number of request tokens has exceeded your per-minute rate limit"}}`, KindLimited},
		{"anthropic", "429 tier spend cap", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"You have reached your API usage limits: your organization has crossed its monthly API usage threshold.","details":{"error_code":"enforced_spend_limit_reached"}}}`, KindLimited},
		{"anthropic", "400 org spend limit", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"You have reached your specified API usage limits. You will regain access on 2026-10-01 at 00:00 UTC."}}`, KindLimited},
		{"anthropic", "400 workspace spend limit", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"You have reached your specified workspace API usage limits. You will regain access on 2026-10-01 at 00:00 UTC."}}`, KindLimited},
		{"anthropic", "400 credit balance", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."}}`, KindLimited},
		{"anthropic", "400 malformed", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"messages: at least one message is required"}}`, KindUnavailable},
		{"anthropic", "404 not_found_error", 404, `{"type":"error","error":{"type":"not_found_error","message":"model: claude-nope"}}`, KindUnavailable},
		{"anthropic", "500 api_error", 500, `{"type":"error","error":{"type":"api_error","message":"Internal server error"}}`, KindUnavailable},
		{"anthropic", "529 overloaded_error", 529, `{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}`, KindUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.provider+"/"+tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			var p Provider = NewOpenAIProvider(srv.URL, srv.Client())
			if tc.provider == "anthropic" {
				p = NewAnthropicProvider(srv.URL, srv.Client())
			}
			_, _, err := collect(t, p, "sk-x")
			assertKind(t, err, tc.want, tc.status)
			// The provider's free-text message (which can echo a masked key) never leaks
			// into the error string that gets logged.
			if strings.Contains(err.Error(), "sk-abc") || strings.Contains(err.Error(), "message") {
				t.Fatalf("error string leaks provider text: %q", err.Error())
			}
		})
	}
}

// assertKind checks err is a *ProviderError of kind want (and status), and that the
// sentinels match exactly that kind.
func assertKind(t *testing.T, err error, want ProviderErrorKind, status int) {
	t.Helper()
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v (%T), want *ProviderError", err, err)
	}
	if pe.Kind != want || pe.Status != status {
		t.Fatalf("kind/status = %s/%d, want %s/%d (%v)", pe.Kind, pe.Status, want, status, err)
	}
	if errors.Is(err, ErrProviderAuth) != (want == KindAuth) {
		t.Fatalf("errors.Is(ErrProviderAuth) = %v for kind %s", !(want == KindAuth), want)
	}
	if errors.Is(err, ErrProviderLimited) != (want == KindLimited) {
		t.Fatalf("errors.Is(ErrProviderLimited) = %v for kind %s", !(want == KindLimited), want)
	}
}

// sseServer serves a 200 event stream with the given raw frames.
func sseServer(frames ...string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, f := range frames {
			_, _ = w.Write([]byte(f + "\n\n"))
		}
	}))
}

func TestOpenAIInlineErrorEvent(t *testing.T) {
	// An error frame after a 200 must surface as a classified error, never a silent
	// truncated success.
	for _, tc := range []struct {
		name  string
		frame string
		want  ProviderErrorKind
	}{
		{"auth", `data: {"error":{"message":"bad key","type":"authentication_error"}}`, KindAuth},
		{"invalid key", `data: {"error":{"message":"bad key","type":"invalid_request_error","code":"invalid_api_key"}}`, KindAuth},
		{"quota", `data: {"error":{"message":"quota","type":"insufficient_quota","code":"insufficient_quota"}}`, KindLimited},
		{"permission", `data: {"error":{"message":"no access","type":"permission_error"}}`, KindLimited},
		{"rate limit", `data: {"error":{"message":"slow down","type":"requests","code":"rate_limit_exceeded"}}`, KindLimited},
		{"server", `data: {"error":{"message":"boom","type":"server_error"}}`, KindUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := sseServer(`data: {"choices":[{"delta":{"content":"Hi"}}]}`, tc.frame)
			defer srv.Close()
			_, _, err := collect(t, NewOpenAIProvider(srv.URL, srv.Client()), "sk")
			assertKind(t, err, tc.want, 0)
		})
	}
}

func TestAnthropicInlineErrorEvent(t *testing.T) {
	for _, tc := range []struct {
		name string
		typ  string
		msg  string
		want ProviderErrorKind
	}{
		{"auth", "authentication_error", "bad key", KindAuth},
		{"billing", "billing_error", "billing", KindLimited},
		{"rate limit", "rate_limit_error", "slow down", KindLimited},
		{"spend limit", "invalid_request_error", "You have reached your specified API usage limits.", KindLimited},
		{"overloaded", "overloaded_error", "Overloaded", KindUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := sseServer(`event: error` + "\n" + `data: {"type":"error","error":{"type":"` + tc.typ + `","message":"` + tc.msg + `"}}`)
			defer srv.Close()
			_, _, err := collect(t, NewAnthropicProvider(srv.URL, srv.Client()), "sk")
			assertKind(t, err, tc.want, 0)
		})
	}
}

func TestStreamTruncation(t *testing.T) {
	t.Run("openai finish_reason length", func(t *testing.T) {
		srv := sseServer(`data: {"choices":[{"delta":{"content":"Hi"}}]}`, `data: {"choices":[{"delta":{},"finish_reason":"length"}]}`, `data: [DONE]`)
		defer srv.Close()
		got, res, err := collect(t, NewOpenAIProvider(srv.URL, srv.Client()), "sk")
		if err != nil || got != "Hi" || !res.Truncated || res.StopReason != "length" {
			t.Fatalf("got %q %+v %v, want Hi truncated(length)", got, res, err)
		}
	})
	t.Run("openai eof before DONE", func(t *testing.T) {
		srv := sseServer(`data: {"choices":[{"delta":{"content":"Hi"}}]}`)
		defer srv.Close()
		_, res, err := collect(t, NewOpenAIProvider(srv.URL, srv.Client()), "sk")
		if err != nil || !res.Truncated {
			t.Fatalf("result %+v %v, want truncated", res, err)
		}
	})
	for stop, want := range map[string]bool{"end_turn": false, "stop_sequence": false, "max_tokens": true, "model_context_window_exceeded": true} {
		t.Run("anthropic "+stop, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { anthropicFrames(w, stop, true) }))
			defer srv.Close()
			got, res, err := collect(t, NewAnthropicProvider(srv.URL, srv.Client()), "sk")
			if err != nil || got != "Hi there" || res.StopReason != stop || res.Truncated != want {
				t.Fatalf("got %q %+v %v, want truncated=%v", got, res, err, want)
			}
		})
	}
	t.Run("anthropic eof before message_stop", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { anthropicFrames(w, "end_turn", false) }))
		defer srv.Close()
		_, res, err := collect(t, NewAnthropicProvider(srv.URL, srv.Client()), "sk")
		if err != nil || !res.Truncated {
			t.Fatalf("result %+v %v, want truncated", res, err)
		}
	})
}
