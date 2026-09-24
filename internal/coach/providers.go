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

// ErrProviderAuth matches (errors.Is) a ProviderError of KindAuth: the upstream provider
// rejected the user's key itself. The service flips enabled=false and routes the user
// back to Settings (ADR-0007). It is the ONLY provider failure that disables a key.
var ErrProviderAuth = errors.New("coach: provider rejected the API key")

// ErrProviderLimited matches (errors.Is) a ProviderError of KindLimited: the key is valid
// but the provider account can't serve the request right now (out of credit, a billing
// problem, a spend/quota/rate limit, or no access to the model/region). The key stays
// enabled; the learner tops up or waits and retries.
var ErrProviderLimited = errors.New("coach: provider account is out of credit or limited")

// ProviderErrorKind classifies a provider failure by what the learner has to do about it.
type ProviderErrorKind int

const (
	// KindUnavailable is a transient or unrecognised upstream failure (5xx, overloaded,
	// timeout, a malformed request). Nothing is wrong with the key or the account.
	KindUnavailable ProviderErrorKind = iota
	// KindAuth is a rejected key: HTTP 401, OpenAI `invalid_api_key`, or an
	// `authentication_error` type. Disables the stored key.
	KindAuth
	// KindLimited is a valid key on an account that is out of credit or limited. Keeps
	// the stored key enabled.
	KindLimited
)

func (k ProviderErrorKind) String() string {
	switch k {
	case KindAuth:
		return "auth"
	case KindLimited:
		return "limited"
	default:
		return "unavailable"
	}
}

// ProviderError is the typed failure a Provider returns when the upstream rejects a
// request, whether as a non-2xx status or as an in-stream error frame after a 200. It
// holds only classification data (the HTTP status and the provider's error type/code
// enum), never the key or the provider's free-text message: OpenAI's invalid-key message
// echoes a partially masked key, so the message is read for classification and dropped.
type ProviderError struct {
	Provider string
	Kind     ProviderErrorKind
	Status   int    // upstream HTTP status; 0 for an in-stream error frame
	Code     string // provider error type/code, e.g. "insufficient_quota", "billing_error"
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("coach: %s provider error (kind=%s status=%d code=%q)", e.Provider, e.Kind, e.Status, e.Code)
}

// Is makes errors.Is(err, ErrProviderAuth) / errors.Is(err, ErrProviderLimited) match
// the corresponding kind.
func (e *ProviderError) Is(target error) bool {
	switch target {
	case ErrProviderAuth:
		return e.Kind == KindAuth
	case ErrProviderLimited:
		return e.Kind == KindLimited
	}
	return false
}

// maxProviderTokens caps a single coach reply on both providers. It is a hard limit on
// TOTAL output, and current models spend part of it thinking (Anthropic adaptive thinking
// and OpenAI reasoning tokens both count), so it has to leave room for the visible reply
// on top of that. The old 1024 cut replies short.
const maxProviderTokens = 4096

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

// StreamResult describes how a stream that ended without an error finished.
type StreamResult struct {
	// StopReason is the provider's raw stop/finish reason ("end_turn", "max_tokens",
	// "stop", "length", …), or "" when the stream never reported one.
	StopReason string
	// Truncated reports the reply was cut off before the model finished: it hit the
	// output cap (Anthropic max_tokens / model_context_window_exceeded, OpenAI length),
	// or the stream ended before its terminal frame.
	Truncated bool
}

// Provider is the provider-agnostic coach interface (ADR-0007's "Coach" interface):
// stream a chat completion using the DECRYPTED user key, delivering each text delta to
// sink. Implementations must not log the key. A sink error (e.g. the client
// disconnected) aborts the stream and is returned. An upstream rejection is a
// *ProviderError.
type Provider interface {
	// Stream calls the provider with apiKey and req, invoking sink for each text delta,
	// and reports how the reply ended.
	Stream(ctx context.Context, apiKey string, req ChatRequest, sink func(delta string) error) (StreamResult, error)
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

func (p *OpenAIProvider) DefaultModel() string { return "gpt-5.6-sol" }

func (p *OpenAIProvider) Stream(ctx context.Context, apiKey string, req ChatRequest, sink func(string) error) (StreamResult, error) {
	msgs := make([]map[string]string, 0, len(req.Messages)+1)
	if req.System != "" {
		msgs = append(msgs, map[string]string{"role": "system", "content": req.System})
	}
	for _, m := range req.Messages {
		msgs = append(msgs, map[string]string{"role": m.Role, "content": m.Content})
	}
	payload := map[string]any{
		"model":  req.Model,
		"stream": true,
		// max_completion_tokens (not the deprecated max_tokens, which reasoning models
		// reject) bounds visible + reasoning tokens together.
		"max_completion_tokens": maxProviderTokens,
		"messages":              msgs,
	}
	if openAISupportsReasoningEffort(req.Model) {
		payload["reasoning_effort"] = openAIReasoningEffort
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return StreamResult{}, fmt.Errorf("coach: build openai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpc.Do(httpReq)
	if err != nil {
		return StreamResult{}, fmt.Errorf("coach: openai request failed: %w", err)
	}
	defer resp.Body.Close()
	if err := providerStatusError("openai", resp, classifyOpenAI); err != nil {
		return StreamResult{}, err
	}

	var res StreamResult
	finished, err := scanSSE(resp.Body, func(data string) (bool, error) {
		if data == "[DONE]" {
			return true, nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
			Error *providerErrorBody `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return false, nil // ignore keep-alive / non-JSON frames
		}
		// An inline error frame (OpenAI can emit `data: {"error":{...}}` after a 200 and
		// close). Surface it instead of presenting a truncated reply as success.
		if chunk.Error != nil {
			return false, chunk.Error.providerError("openai", 0, classifyOpenAI)
		}
		if len(chunk.Choices) > 0 {
			c := chunk.Choices[0]
			if c.Delta.Content != "" {
				if err := sink(c.Delta.Content); err != nil {
					return false, err
				}
			}
			if c.FinishReason != "" {
				res.StopReason = c.FinishReason
			}
		}
		return false, nil
	})
	if err != nil {
		return StreamResult{}, err
	}
	res.Truncated = !finished || res.StopReason == "length"
	return res, nil
}

// openAIReasoningEffort is the reasoning effort the coach asks for. gpt-5.6 defaults to
// medium, and reasoning tokens count toward max_completion_tokens, so a higher effort
// could spend the cap before any visible text.
const openAIReasoningEffort = "low"

// openAISupportsReasoningEffort reports whether model takes `reasoning_effort` on Chat
// Completions: the gpt-5 and gpt-6 reasoning families, which all accept "low" per their
// model pages (developers.openai.com/api/docs/models, 2026-09-24). Non-reasoning models
// (gpt-4o, gpt-4.1, the *-chat variants) answer it with a 400, and Settings accepts any
// model id, so it is sent only to these.
func openAISupportsReasoningEffort(model string) bool {
	return (strings.HasPrefix(model, "gpt-5") || strings.HasPrefix(model, "gpt-6")) && !strings.Contains(model, "-chat")
}

// classifyOpenAI maps an OpenAI failure to a kind, per
// developers.openai.com/api/docs/guides/error-codes plus the codes real error bodies
// carry. Only a rejected key disables it: 401, `invalid_api_key`, or an
// `authentication_error` type. These keep the key: out of credit or quota (429
// `insufficient_quota` / `credit_balance_exhausted`), spend and usage limits (429
// `organization_spend_limit_exceeded` / `project_spend_limit_exceeded` /
// `organization_usage_limit_exceeded`, the legacy 400 `billing_hard_limit_reached`),
// rate limits (429), a 402, and model or region permission errors (403
// `unsupported_country_region_territory`, 403/404 `model_not_found`).
func classifyOpenAI(status int, typ, code, _ string) ProviderErrorKind {
	switch {
	case status == http.StatusUnauthorized || code == "invalid_api_key" || typ == "authentication_error":
		return KindAuth
	case status == http.StatusPaymentRequired || status == http.StatusForbidden || status == http.StatusTooManyRequests:
		return KindLimited
	}
	switch code {
	case "insufficient_quota", "credit_balance_exhausted",
		"organization_spend_limit_exceeded", "project_spend_limit_exceeded", "organization_usage_limit_exceeded",
		"billing_hard_limit_reached", "billing_not_active",
		"rate_limit_exceeded", "slow_down",
		"unsupported_country_region_territory", "model_not_found":
		return KindLimited
	}
	switch typ {
	case "insufficient_quota", "billing_error", "rate_limit_error", "permission_error", "requests", "tokens":
		return KindLimited
	}
	return KindUnavailable
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

func (p *AnthropicProvider) DefaultModel() string { return "claude-sonnet-5" }

// anthropicEffort is the effort level the coach asks for. Current models default to
// high effort and think adaptively, and thinking counts toward max_tokens; a chat coach
// wants short, quick replies, which is what low effort is for.
const anthropicEffort = "low"

// anthropicEffortModels are the model ids that accept `output_config.effort` (the GA
// effort parameter, no beta header), per platform.claude.com/docs/en/build-with-claude/effort
// as of 2026-09-24. Any other model (e.g. claude-haiku-4-5) answers a request carrying it
// with a 400 invalid_request_error, and Settings lets a learner type any model id, so the
// field is sent only to these.
var anthropicEffortModels = []string{
	"claude-fable-5-1", "claude-mythos-5-1", "claude-fable-5", "claude-mythos-5", "claude-mythos-preview",
	"claude-opus-5-5", "claude-opus-5", "claude-opus-4-8", "claude-opus-4-7", "claude-opus-4-6", "claude-opus-4-5",
	"claude-sonnet-5", "claude-sonnet-4-6",
}

// anthropicSupportsEffort reports whether model accepts output_config.effort: an exact
// match, or a dated snapshot of one (e.g. claude-opus-4-5-20251101).
func anthropicSupportsEffort(model string) bool {
	for _, m := range anthropicEffortModels {
		if model == m || strings.HasPrefix(model, m+"-") {
			return true
		}
	}
	return false
}

func (p *AnthropicProvider) Stream(ctx context.Context, apiKey string, req ChatRequest, sink func(string) error) (StreamResult, error) {
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
	if anthropicSupportsEffort(req.Model) {
		payload["output_config"] = map[string]any{"effort": anthropicEffort}
	}
	if req.System != "" {
		payload["system"] = req.System
	}
	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return StreamResult{}, fmt.Errorf("coach: build anthropic request: %w", err)
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpc.Do(httpReq)
	if err != nil {
		return StreamResult{}, fmt.Errorf("coach: anthropic request failed: %w", err)
	}
	defer resp.Body.Close()
	if err := providerStatusError("anthropic", resp, classifyAnthropic); err != nil {
		return StreamResult{}, err
	}

	var res StreamResult
	finished, err := scanSSE(resp.Body, func(data string) (bool, error) {
		var ev struct {
			Type  string `json:"type"`
			Delta struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
			Error providerErrorBody `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return false, nil
		}
		switch ev.Type {
		case "content_block_delta":
			// Only visible text; thinking/signature deltas carry no `text` and stay private.
			if ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				if err := sink(ev.Delta.Text); err != nil {
					return false, err
				}
			}
		case "message_delta":
			// The stop reason arrives here (null on message_start, absent elsewhere).
			if ev.Delta.StopReason != "" {
				res.StopReason = ev.Delta.StopReason
			}
		case "error":
			return false, ev.Error.providerError("anthropic", 0, classifyAnthropic)
		case "message_stop":
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return StreamResult{}, err
	}
	res.Truncated = !finished || res.StopReason == "max_tokens" || res.StopReason == "model_context_window_exceeded"
	return res, nil
}

// classifyAnthropic maps an Anthropic failure to a kind, per
// platform.claude.com/docs/en/api/errors and /api/rate-limits. Only a rejected key
// (401 / `authentication_error`) disables it. These keep the key: 402 `billing_error`
// (billing or credit), 403 `permission_error` (no access to the resource or model),
// 429 `rate_limit_error` (a rate limit, or the tier's monthly spend cap), and the 400
// `invalid_request_error` a self-set org/workspace spend limit returns. That last one
// can only be told apart from a malformed request by its message ("You have reached your
// specified [workspace] API usage limits…"); older accounts saw "Your credit balance is
// too low…" on a 400 too.
func classifyAnthropic(status int, typ, _, message string) ProviderErrorKind {
	switch {
	case status == http.StatusUnauthorized || typ == "authentication_error":
		return KindAuth
	case status == http.StatusPaymentRequired || typ == "billing_error",
		status == http.StatusForbidden || typ == "permission_error",
		status == http.StatusTooManyRequests || typ == "rate_limit_error":
		return KindLimited
	case typ == "invalid_request_error" && anthropicSpendLimitMessage(message):
		return KindLimited
	}
	return KindUnavailable
}

// anthropicSpendLimitMessage reports whether an invalid_request_error message is a spend
// limit or credit-balance rejection rather than a malformed request.
func anthropicSpendLimitMessage(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "api usage limits") || strings.Contains(m, "credit balance")
}

// --- shared error + SSE plumbing ---

// classifyFunc maps an upstream failure (HTTP status, 0 for an in-stream frame; the
// provider's error type, code, and message) to a ProviderErrorKind.
type classifyFunc func(status int, typ, code, message string) ProviderErrorKind

// providerErrorBody is the `error` object both providers use in error bodies and
// in-stream error frames. `code` is decoded loosely because OpenAI sends a string or null.
type providerErrorBody struct {
	Type    string          `json:"type"`
	Code    json.RawMessage `json:"code"`
	Message string          `json:"message"`
}

// providerError classifies the body into a *ProviderError. The message is used only to
// classify and is not kept.
func (b *providerErrorBody) providerError(provider string, status int, classify classifyFunc) *ProviderError {
	var code string
	_ = json.Unmarshal(b.Code, &code) // a non-string code (null/number) stays ""
	kind := classify(status, b.Type, code, b.Message)
	label := code
	if label == "" {
		label = b.Type
	}
	return &ProviderError{Provider: provider, Kind: kind, Status: status, Code: label}
}

// scanSSE reads an event-stream body line by line and hands each `data:` payload to fn,
// stopping when fn reports done (true) or returns an error. It reports whether fn
// signalled done, so a caller can tell a complete stream from one that hit EOF early.
// Blank lines / comment (`:`) lines / `event:` lines are skipped. The scanner buffer is
// enlarged so a long single frame does not overflow bufio's default 64KiB line cap.
func scanSSE(body io.Reader, fn func(data string) (done bool, err error)) (bool, error) {
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
			return false, err
		}
		if done {
			return true, nil
		}
	}
	return false, sc.Err()
}

// providerStatusError turns a non-2xx response into a classified *ProviderError. It reads
// up to 8KiB of the body for the provider's error type/code (and, for Anthropic, the
// message) so it can tell a rejected key from an out-of-credit account; nothing from the
// body is echoed or logged.
func providerStatusError(provider string, resp *http.Response, classify classifyFunc) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
	var env struct {
		Error providerErrorBody `json:"error"`
	}
	_ = json.Unmarshal(raw, &env) // an unparseable body classifies on the status alone
	return env.Error.providerError(provider, resp.StatusCode, classify)
}
