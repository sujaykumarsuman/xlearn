package coach

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// ErrProviderAuth is returned by a Provider when the user's key is rejected by the
// upstream provider (HTTP 401/403). The service flips enabled=false and routes the user
// back to Settings (ADR-0007). It never carries the key or the provider body verbatim.
var ErrProviderAuth = errors.New("coach: provider rejected the API key")

// maxProviderTokens caps a single coach reply (Anthropic requires max_tokens; we apply
// the same ceiling to keep replies bounded and cheap on the user's key).
const maxProviderTokens = 1024

// ChatMessage is one prior turn sent to the provider (role user|assistant).
type ChatMessage struct {
	Role    string
	Content string
}

// ChatRequest is a provider-agnostic streaming chat request. System is the
// server-built page-context prompt (Socratic vs reviewer); Messages are the thread's
// turns oldest-first ending with the new user message.
type ChatRequest struct {
	Model    string
	System   string
	Messages []ChatMessage
}

// Provider is the provider-agnostic coach interface (ADR-0007's "Coach" interface):
// stream a chat completion using the DECRYPTED user key, delivering each text delta to
// sink. Implementations must not log the key. A sink error (e.g. the client
// disconnected) aborts the stream and is returned. Auth rejection returns ErrProviderAuth.
type Provider interface {
	// Stream calls the provider with apiKey and req, invoking sink for each text delta.
	Stream(ctx context.Context, apiKey string, req ChatRequest, sink func(delta string) error) error
	// DefaultModel is the fallback model id when the account set none.
	DefaultModel() string
}

// providerFor returns the Provider implementation for a stored provider id.
func (s *Service) providerFor(id string) (Provider, bool) {
	switch id {
	case store.ProviderOpenAI:
		return s.openai, true
	case store.ProviderAnthropic:
		return s.anthropic, true
	default:
		return nil, false
	}
}

// --- OpenAI (chat completions, streaming) ---

// OpenAIProvider streams from OpenAI's /v1/chat/completions API.
type OpenAIProvider struct {
	baseURL string
	httpc   *http.Client
}

// NewOpenAIProvider builds the OpenAI provider. httpc should have NO overall timeout
// (streaming); per-call bounds come from the request context.
func NewOpenAIProvider(baseURL string, httpc *http.Client) *OpenAIProvider {
	return &OpenAIProvider{baseURL: strings.TrimRight(baseURL, "/"), httpc: httpc}
}

func (p *OpenAIProvider) DefaultModel() string { return "gpt-4o-mini" }

func (p *OpenAIProvider) Stream(ctx context.Context, apiKey string, req ChatRequest, sink func(string) error) error {
	msgs := make([]map[string]string, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	body, _ := json.Marshal(map[string]any{
		"model":    req.Model,
		"stream":   true,
		"messages": msgs,
	})
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("coach: build openai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpc.Do(httpReq)
	if err != nil {
		return fmt.Errorf("coach: openai request failed: %w", err)
	}
	defer resp.Body.Close()
	if err := providerStatusError("openai", resp); err != nil {
		return err
	}

	return scanSSE(resp.Body, func(data string) (bool, error) {
		if data == "[DONE]" {
			return true, nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
			Error *struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return false, nil // ignore keep-alive / non-JSON frames
		}
		// An inline error frame (OpenAI can emit `data: {"error":{...}}` after a 200 and
		// close). Surface it instead of presenting a truncated reply as success; map an
		// auth/permission error to ErrProviderAuth so the key is disabled.
		if chunk.Error != nil {
			if chunk.Error.Code == "invalid_api_key" || chunk.Error.Type == "authentication_error" || chunk.Error.Type == "permission_error" || chunk.Error.Type == "insufficient_quota" {
				return false, ErrProviderAuth
			}
			return false, fmt.Errorf("coach: openai stream error: %s", chunk.Error.Type)
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			if err := sink(chunk.Choices[0].Delta.Content); err != nil {
				return false, err
			}
		}
		return false, nil
	})
}

// --- Anthropic (messages, streaming) ---

// AnthropicProvider streams from Anthropic's /v1/messages API.
type AnthropicProvider struct {
	baseURL string
	httpc   *http.Client
}

// NewAnthropicProvider builds the Anthropic provider.
func NewAnthropicProvider(baseURL string, httpc *http.Client) *AnthropicProvider {
	return &AnthropicProvider{baseURL: strings.TrimRight(baseURL, "/"), httpc: httpc}
}

func (p *AnthropicProvider) DefaultModel() string { return "claude-3-5-sonnet-latest" }

func (p *AnthropicProvider) Stream(ctx context.Context, apiKey string, req ChatRequest, sink func(string) error) error {
	msgs := make([]map[string]string, 0, len(req.Messages))
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	payload := map[string]any{
		"model":      req.Model,
		"max_tokens": maxProviderTokens,
		"stream":     true,
		"messages":   msgs,
	}
	if req.System != "" {
		payload["system"] = req.System
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("coach: build anthropic request: %w", err)
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpc.Do(httpReq)
	if err != nil {
		return fmt.Errorf("coach: anthropic request failed: %w", err)
	}
	defer resp.Body.Close()
	if err := providerStatusError("anthropic", resp); err != nil {
		return err
	}

	return scanSSE(resp.Body, func(data string) (bool, error) {
		var ev struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Error struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return false, nil
		}
		switch ev.Type {
		case "content_block_delta":
			if ev.Delta.Text != "" {
				if err := sink(ev.Delta.Text); err != nil {
					return false, err
				}
			}
		case "error":
			if ev.Error.Type == "authentication_error" || ev.Error.Type == "permission_error" {
				return false, ErrProviderAuth
			}
			return false, fmt.Errorf("coach: anthropic stream error: %s", ev.Error.Type)
		case "message_stop":
			return true, nil
		}
		return false, nil
	})
}

// --- shared SSE plumbing ---

// scanSSE reads an event-stream body line by line and hands each `data:` payload to fn,
// stopping when fn reports done (true) or returns an error. Blank lines / comment
// (`:`) lines / `event:` lines are skipped. The scanner buffer is enlarged so a long
// single frame does not overflow bufio's default 64KiB line cap.
func scanSSE(body io.Reader, fn func(data string) (done bool, err error)) error {
	sc := bufio.NewScanner(body)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(line[len("data:"):])
		if data == "" {
			continue
		}
		done, err := fn(data)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}
	return sc.Err()
}

// providerStatusError inspects a non-streaming error status. 401/403 → ErrProviderAuth
// (key rejected); other non-2xx → a generic error. The provider's response body is NOT
// echoed to avoid leaking anything sensitive; only the status is used.
func providerStatusError(provider string, resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	// Drain a little of the body so the connection can be reused, then discard it.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8<<10))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrProviderAuth
	}
	return fmt.Errorf("coach: %s returned status %d", provider, resp.StatusCode)
}
