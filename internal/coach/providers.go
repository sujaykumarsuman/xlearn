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
	"slices"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// Typed provider failures (t5 §9 ADR-0007 fixes, t5 §10 — T6 shares these). The whole
// point of the taxonomy is that the learner is told what to actually DO, and that
// ONLY a rejected key is ever disabled: v1 disabled a key on any "limited" answer, so
// running out of credit made the learner re-paste a perfectly good key after topping up.
var (
	// ErrProviderAuth matches a ProviderError of KindAuth: the provider rejected the key
	// ITSELF. The service flips enabled=false and routes the learner back to Settings
	// (ADR-0007). It is the ONLY provider failure that disables a key.
	ErrProviderAuth = errors.New("coach: provider rejected the API key")

	// ErrQuota — the account is out of credit, has a billing problem, or hit a spend or
	// usage limit. The key is fine; the learner tops up.
	ErrQuota = errors.New("coach: provider account is out of credit or over a spend limit")
	// ErrRateLimited — too many requests or tokens for now. The key is fine; retry later.
	ErrRateLimited = errors.New("coach: provider rate limit reached")
	// ErrModelAccess — the key may not use THIS MODEL (a 403 permission error on the
	// model, or a 404 model-not-found). The key is fine and the fix is to pick another
	// model, which is what AB01 F11 offers. Never a disable: a model that was retired
	// under the learner, or one their org hasn't enabled, says nothing about the key.
	ErrModelAccess = errors.New("coach: key cannot use this model")
	// ErrRegion — the provider does not serve the account's region.
	ErrRegion = errors.New("coach: provider does not serve this region")

	// ErrProviderLimited is the UMBRELLA over every non-auth, account-side failure
	// (quota, rate limit, model access, region). It is kept so v1 call sites that only
	// asked "is this the key's fault or the account's?" keep working unchanged while the
	// finer sentinels above are adopted; it deliberately does NOT match KindUnavailable,
	// which is our problem or the provider's, not the account's.
	ErrProviderLimited = errors.New("coach: provider account is out of credit or limited")
)

// ProviderErrorKind classifies a provider failure by what the learner has to do about it.
type ProviderErrorKind int

const (
	// KindUnavailable is a transient or unrecognised upstream failure (5xx, overloaded,
	// timeout, a malformed request). Nothing is wrong with the key or the account.
	KindUnavailable ProviderErrorKind = iota
	// KindAuth is a rejected key: HTTP 401, OpenAI `invalid_api_key`, or an
	// `authentication_error` type. DISABLES the stored key.
	KindAuth
	// KindQuota is out of credit, a billing failure, or a spend/usage limit. Keeps the key.
	KindQuota
	// KindRateLimited is a request/token rate limit. Keeps the key.
	KindRateLimited
	// KindModelAccess is "this key may not use this model" (403 permission_error on a
	// model, 404 not_found_error / model_not_found). Keeps the key; the learner switches
	// model (AB01 F11).
	KindModelAccess
	// KindRegion is an unsupported country/region/territory. Keeps the key.
	KindRegion
)

// String is the stable wire `reason` the SSE error event and the pre-stream 429 envelope
// carry, which the SPA maps to AB01's copy. Keep these values stable: Coach.tsx switches
// on them, and T6 reuses the same taxonomy (t5 §10).
func (k ProviderErrorKind) String() string {
	switch k {
	case KindAuth:
		return "auth"
	case KindQuota:
		return "quota"
	case KindRateLimited:
		return "rate_limit"
	case KindModelAccess:
		return "model_access"
	case KindRegion:
		return "region"
	default:
		return "unavailable"
	}
}

// AccountSide reports whether the kind is the learner's account's doing (as opposed to a
// bad key, or an upstream outage) — i.e. whether ErrProviderLimited matches it.
func (k ProviderErrorKind) AccountSide() bool {
	switch k {
	case KindQuota, KindRateLimited, KindModelAccess, KindRegion:
		return true
	default:
		return false
	}
}

// ErrorReason extracts the wire `reason` from a provider error, defaulting to
// "unavailable" for anything that isn't a classified *ProviderError (a transport failure,
// a context cancellation).
func ErrorReason(err error) string {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Kind.String()
	}
	return KindUnavailable.String()
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

// Is makes errors.Is match each sentinel against its kind, with ErrProviderLimited
// matching every account-side kind at once.
func (e *ProviderError) Is(target error) bool {
	switch target {
	case ErrProviderAuth:
		return e.Kind == KindAuth
	case ErrQuota:
		return e.Kind == KindQuota
	case ErrRateLimited:
		return e.Kind == KindRateLimited
	case ErrModelAccess:
		return e.Kind == KindModelAccess
	case ErrRegion:
		return e.Kind == KindRegion
	case ErrProviderLimited:
		return e.Kind.AccountSide()
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

// Usage is the token count a stream reported for one turn. HasTokens distinguishes "the
// provider said zero" from "the provider said nothing", which is what decides whether a
// cost estimate is stored at all.
type Usage struct {
	InputTokens  int
	OutputTokens int
	HasTokens    bool
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
	// Usage is what the turn cost in tokens, when the stream reported it. Stored on the
	// assistant message and summed into the month-to-date figure.
	Usage Usage
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

// providerFor returns the Provider implementation for a stored provider id, from the
// Service's REGISTRY. The registry is the single list of providers coach serves: the
// handlers validate an incoming provider against it, GET /models enumerates it, and
// store.ValidProvider is only a static mirror of the DB CHECK (store cannot import coach
// without a cycle). A test asserts registry, mirror and CHECK agree, so adding a provider
// in one place and forgetting the others fails CI rather than production.
func (s *Service) providerFor(id string) (Provider, bool) {
	p, ok := s.providers[id]
	return p, ok
}

// ProviderIDs returns the registry's provider ids in a stable order (the order GET /models
// lists them, and the order the onboarding step offers them).
func (s *Service) ProviderIDs() []string {
	out := make([]string, 0, len(s.providers))
	for _, id := range providerOrder {
		if _, ok := s.providers[id]; ok {
			out = append(out, id)
		}
	}
	// Anything registered that the canonical order doesn't know about still gets listed,
	// so a new provider can never be silently invisible in the UI.
	for id := range s.providers {
		if !slices.Contains(out, id) {
			out = append(out, id)
		}
	}
	return out
}

// providerOrder is the canonical display order of the provider registry.
var providerOrder = []string{store.ProviderAnthropic, store.ProviderOpenAI}

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

// DefaultModel is the catalog's OpenAI default (OpenAIDefaultModel documents why this
// release keeps gpt-5.6-sol rather than promoting gpt-6-sol).
func (p *OpenAIProvider) DefaultModel() string { return OpenAIDefaultModel }

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
		// store:false — do NOT let the provider retain this conversation for its own
		// dashboards or training-adjacent features. v1 left it at OpenAI's default (true),
		// which silently logged every learner's coach conversation to the learner's own
		// OpenAI org. These are study conversations about problems the learner is being
		// assessed on, and xLearn promises the key is used for nothing but the call, so
		// the retention default has to be flipped explicitly on EVERY request. A test
		// asserts it on every request the provider makes, not just the happy path.
		//
		// (An org with Zero Data Retention enabled is treated as store:false regardless,
		// so this never conflicts with a stricter org policy.)
		"store": false,
		// Ask for the usage frame: with include_usage the stream emits one final chunk
		// carrying the whole request's token counts before [DONE]. Without it a streamed
		// completion reports no usage at all and every turn's cost would be unknown.
		"stream_options": map[string]any{"include_usage": true},
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
			// The include_usage frame: a final chunk with empty choices and the whole
			// request's counts. A pointer so an absent `usage` is distinguishable from a
			// zeroed one.
			Usage *struct {
				PromptTokens     int `json:"prompt_tokens"`
				CompletionTokens int `json:"completion_tokens"`
			} `json:"usage"`
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
		if chunk.Usage != nil {
			res.Usage = Usage{
				InputTokens:  chunk.Usage.PromptTokens,
				OutputTokens: chunk.Usage.CompletionTokens,
				HasTokens:    true,
			}
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
// carry.
//
// Order matters: the provider's own error CODE is the most specific signal, then its
// TYPE, then the bare HTTP STATUS. Classifying on status first (as v1 did) collapses
// distinct outcomes — a 429 is a rate limit OR an exhausted quota, and a 403 is a model
// permission error OR a blocked region — and the learner-facing advice differs for each.
//
// Only a rejected key disables it: 401, `invalid_api_key`, or `authentication_error`.
func classifyOpenAI(status int, typ, code, message string) ProviderErrorKind {
	switch code {
	case "invalid_api_key":
		return KindAuth
	case "unsupported_country_region_territory":
		return KindRegion
	case "model_not_found":
		return KindModelAccess
	case "insufficient_quota", "credit_balance_exhausted",
		"organization_spend_limit_exceeded", "project_spend_limit_exceeded", "organization_usage_limit_exceeded",
		"billing_hard_limit_reached", "billing_not_active":
		return KindQuota
	case "rate_limit_exceeded", "slow_down":
		return KindRateLimited
	}
	switch typ {
	case "authentication_error":
		return KindAuth
	case "insufficient_quota", "billing_error":
		return KindQuota
	// "requests" and "tokens" are the rate-limit buckets OpenAI names in `type`.
	case "rate_limit_error", "requests", "tokens":
		return KindRateLimited
	case "not_found_error":
		return KindModelAccess
	case "permission_error":
		if regionMessage(message) {
			return KindRegion
		}
		return KindModelAccess
	}
	switch status {
	case http.StatusUnauthorized:
		return KindAuth
	case http.StatusPaymentRequired:
		return KindQuota
	case http.StatusForbidden, http.StatusNotFound:
		if regionMessage(message) {
			return KindRegion
		}
		return KindModelAccess
	case http.StatusTooManyRequests:
		return KindRateLimited
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

// DefaultModel is the catalog's Anthropic default.
func (p *AnthropicProvider) DefaultModel() string { return AnthropicDefaultModel }

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
			// Anthropic splits usage across two frames: input_tokens arrives up front on
			// message_start.message.usage, output_tokens accumulates and is final on
			// message_delta.usage.
			Message struct {
				Usage *struct {
					InputTokens  int `json:"input_tokens"`
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			} `json:"message"`
			Usage *struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
			Error providerErrorBody `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			return false, nil
		}
		switch ev.Type {
		case "message_start":
			if u := ev.Message.Usage; u != nil {
				res.Usage.InputTokens = u.InputTokens
				res.Usage.HasTokens = true
				// message_start may already carry a non-zero output count; keep it rather
				// than assuming zero, in case no message_delta usage frame follows.
				if u.OutputTokens > 0 {
					res.Usage.OutputTokens = u.OutputTokens
				}
			}
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
			// …and the final output count, which supersedes anything seen so far. An
			// input_tokens here (Anthropic sends it on some paths) fills in a stream that
			// never reported one on message_start.
			if u := ev.Usage; u != nil {
				res.Usage.OutputTokens = u.OutputTokens
				res.Usage.HasTokens = true
				if u.InputTokens > 0 && res.Usage.InputTokens == 0 {
					res.Usage.InputTokens = u.InputTokens
				}
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
// platform.claude.com/docs/en/api/errors and /api/rate-limits. Type first, status second,
// message last — same reasoning as classifyOpenAI.
//
// Only a rejected key (401 / `authentication_error`) disables it. These keep the key:
// 402 `billing_error` (billing or credit) → quota; 403 `permission_error` and 404
// `not_found_error` (no access to the model) → model access; 429 `rate_limit_error` → rate
// limit; and the 400 `invalid_request_error` a self-set org/workspace spend limit returns
// → quota. That last one can only be told apart from a genuinely malformed request by its
// message ("You have reached your specified [workspace] API usage limits…"); older
// accounts saw "Your credit balance is too low…" on a 400 too. Hence
// anthropicSpendLimitMessage: dropping it would turn a top-up-and-retry into "the coach is
// broken".
func classifyAnthropic(status int, typ, _, message string) ProviderErrorKind {
	switch typ {
	case "authentication_error":
		return KindAuth
	case "billing_error":
		return KindQuota
	case "rate_limit_error":
		// Anthropic answers BOTH a per-minute rate limit and the tier's monthly spend cap
		// with 429 rate_limit_error, and only the message separates them. The advice is
		// opposite — "wait a moment" versus "top up" — so the spend-cap wording wins.
		if anthropicSpendLimitMessage(message) {
			return KindQuota
		}
		return KindRateLimited
	case "not_found_error":
		return KindModelAccess
	case "permission_error":
		// Anthropic returns 403 permission_error for a region block as well as for a
		// model the account may not use; only the message separates them.
		if regionMessage(message) {
			return KindRegion
		}
		return KindModelAccess
	case "invalid_request_error":
		if anthropicSpendLimitMessage(message) {
			return KindQuota
		}
		return KindUnavailable
	}
	switch status {
	case http.StatusUnauthorized:
		return KindAuth
	case http.StatusPaymentRequired:
		return KindQuota
	case http.StatusForbidden, http.StatusNotFound:
		if regionMessage(message) {
			return KindRegion
		}
		return KindModelAccess
	case http.StatusTooManyRequests:
		return KindRateLimited
	}
	return KindUnavailable
}

// anthropicSpendLimitMessage reports whether an invalid_request_error message is a spend
// limit or credit-balance rejection rather than a malformed request.
func anthropicSpendLimitMessage(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "api usage limits") || strings.Contains(m, "credit balance")
}

// regionMessage reports whether a 403/404 message is about an unsupported country, region
// or territory rather than the model. Both providers answer a geo block with the same
// status they use for a model permission error, and only the prose distinguishes them —
// "top up" / "pick another model" advice would both be wrong for a region block.
func regionMessage(message string) bool {
	m := strings.ToLower(message)
	return strings.Contains(m, "country, region, or territory") ||
		strings.Contains(m, "unsupported_country_region_territory") ||
		strings.Contains(m, "not available in your region") ||
		strings.Contains(m, "unsupported region")
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
