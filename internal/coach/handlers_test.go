package coach

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/httpx"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// harness wires a coach Service (mem store + fixed-claims verifier + real cipher) behind
// the standard middleware chain, plus a swappable fake provider server.
type harness struct {
	store    *memStore
	cipher   *secrets.Cipher
	server   *httptest.Server
	provider *httptest.Server
	account  string

	// provHandler is the current fake provider behaviour (set per test).
	provHandler http.HandlerFunc
	// lastAuth captures the Authorization / x-api-key the provider saw (the decrypted key).
	lastAuth string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{store: newMemStore(), cipher: testCipher(), account: "11111111-1111-4111-8111-111111111111"}

	h.provider = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a := r.Header.Get("Authorization"); a != "" {
			h.lastAuth = strings.TrimPrefix(a, "Bearer ")
		} else {
			h.lastAuth = r.Header.Get("x-api-key")
		}
		if h.provHandler != nil {
			h.provHandler(w, r)
			return
		}
		openAISSE(w)
	}))
	t.Cleanup(h.provider.Close)

	openai := NewOpenAIProvider(h.provider.URL, h.provider.Client())
	anthropic := NewAnthropicProvider(h.provider.URL, h.provider.Client())
	svc := NewService(h.store, fakeVerifier{subject: h.account}, h.cipher, openai, anthropic, discardLogger())

	handler := httpx.Chain(svc.Handler(), httpx.RequestID, httpx.AccessLog(discardLogger()), httpx.Recoverer(discardLogger()))
	h.server = httptest.NewServer(handler)
	t.Cleanup(h.server.Close)
	return h
}

func (h *harness) do(t *testing.T, method, path string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	}
	req, _ := http.NewRequestWithContext(context.Background(), method, h.server.URL+path, rdr)
	req.Header.Set("Authorization", "Bearer test-token")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response, v any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

// --- key CRUD ---

func TestPutGetDeleteKey(t *testing.T) {
	h := newHarness(t)
	raw := "sk-openai-abcdefghijklmnop-cdef"

	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "key": raw, "default_model": "gpt-4o-mini"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT status %d", resp.StatusCode)
	}
	var put keysResponse
	decode(t, resp, &put)
	if !put.Connected || len(put.Keys) != 1 {
		t.Fatalf("put response = %+v", put)
	}
	if put.Keys[0].MaskedKey != secrets.Mask(raw) {
		t.Fatalf("masked = %q, want %q", put.Keys[0].MaskedKey, secrets.Mask(raw))
	}
	// The account's first key becomes its default.
	if !put.Keys[0].IsDefault || put.DefaultProvider != "openai" {
		t.Fatalf("first key should be the default: %+v", put)
	}

	// GET returns the masked view — never the raw key.
	resp = h.do(t, http.MethodGet, "/keys", nil, nil)
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.Contains(string(bodyBytes), raw) {
		t.Fatalf("GET /keys leaked the raw key: %s", bodyBytes)
	}
	var got keysResponse
	_ = json.Unmarshal(bodyBytes, &got)
	if !got.Connected || got.Keys[0].Provider != "openai" || !got.Keys[0].Enabled {
		t.Fatalf("get = %s", bodyBytes)
	}

	// The stored material must decrypt back to the raw key (envelope round-trip).
	kc, _ := h.store.GetKey(context.Background(), h.account, "openai")
	dec, err := h.cipher.Open(kc.EncKey, kc.EncDataKey)
	if err != nil || string(dec) != raw {
		t.Fatalf("stored key did not round-trip: dec=%q err=%v", dec, err)
	}

	// DELETE removes it.
	resp = h.do(t, http.MethodDelete, "/keys?provider=openai", nil, nil)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE status %d", resp.StatusCode)
	}
	resp.Body.Close()
	resp = h.do(t, http.MethodGet, "/keys", nil, nil)
	var empty keysResponse
	decode(t, resp, &empty)
	if empty.Connected || len(empty.Keys) != 0 {
		t.Fatalf("after delete = %+v", empty)
	}
}

func TestGetKeyEmptyState(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodGet, "/keys", nil, nil)
	var out keysResponse
	decode(t, resp, &out)
	if out.Connected || len(out.Keys) != 0 {
		t.Fatalf("empty state = %+v", out)
	}
}

func TestPutRejectsBadProvider(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "google", "key": "AIza-xxxxxxxxxxxx"}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status %d, want 422", resp.StatusCode)
	}
}

func TestPutToggleEnabled(t *testing.T) {
	h := newHarness(t)
	h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "key": "sk-openai-abcdefghij-cdef"}, nil).Body.Close()

	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "enabled": false}, nil)
	var off keysResponse
	decode(t, resp, &off)
	if off.Keys[0].Enabled {
		t.Fatal("toggle off did not disable")
	}
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "enabled": true}, nil)
	var on keysResponse
	decode(t, resp, &on)
	if !on.Keys[0].Enabled {
		t.Fatal("toggle on did not enable")
	}
}

func TestToggleWithoutKey404(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "enabled": true}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

// TestPutUpdateMeta covers F006: switching the coach's model + name with NO key in the
// body updates the config in place without re-sealing (the stored secret is untouched).
func TestPutUpdateMeta(t *testing.T) {
	h := newHarness(t)
	raw := "sk-openai-abcdefghijklmnop-cdef"
	h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "key": raw, "default_model": "gpt-5.6-terra", "name": "Terra"}, nil).Body.Close()

	// Meta-only edit: new model + name, no key.
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "default_model": "gpt-5.6-sol", "name": "Deep Thinker"}, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("meta PUT status %d", resp.StatusCode)
	}
	var out keysResponse
	decode(t, resp, &out)
	if len(out.Keys) != 1 || out.Keys[0].DefaultModel != "gpt-5.6-sol" || out.Keys[0].Name != "Deep Thinker" {
		t.Fatalf("meta update = %+v", out)
	}
	if !out.Keys[0].Enabled || out.Keys[0].MaskedKey != secrets.Mask(raw) {
		t.Fatalf("meta update should keep the key enabled + masked: %+v", out)
	}

	// The sealed material must still round-trip to the ORIGINAL raw key (never re-sealed).
	kc, _ := h.store.GetKey(context.Background(), h.account, "openai")
	dec, err := h.cipher.Open(kc.EncKey, kc.EncDataKey)
	if err != nil || string(dec) != raw {
		t.Fatalf("meta update disturbed the sealed key: dec=%q err=%v", dec, err)
	}

	// A partial edit (name only) keeps the current model.
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "name": "Renamed"}, nil)
	var partial keysResponse
	decode(t, resp, &partial)
	if partial.Keys[0].DefaultModel != "gpt-5.6-sol" || partial.Keys[0].Name != "Renamed" {
		t.Fatalf("partial meta update = %+v", partial)
	}
}

func TestUpdateMetaWithoutKey404(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "default_model": "gpt-5.6-sol"}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

// TestMultiProviderAndDefault covers F006 round 2: connect both providers, flip the default,
// and confirm deleting the default promotes the survivor.
func TestMultiProviderAndDefault(t *testing.T) {
	h := newHarness(t)
	// First key (anthropic) becomes the default.
	h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "anthropic", "key": "sk-ant-abcdefghij-4a2f", "default_model": "claude-sonnet-5", "name": "Sonnet 5"}, nil).Body.Close()
	// Second key (openai) — connected, not default.
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "key": "sk-openai-abcdefghij-9f2c", "default_model": "gpt-5.6-sol"}, nil)
	var two keysResponse
	decode(t, resp, &two)
	if len(two.Keys) != 2 || two.DefaultProvider != "anthropic" {
		t.Fatalf("two keys, anthropic default = %+v", two)
	}

	// Flip the default to openai (no key in the body).
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "default": true}, nil)
	var flipped keysResponse
	decode(t, resp, &flipped)
	if flipped.DefaultProvider != "openai" {
		t.Fatalf("set-default did not move: %+v", flipped)
	}
	for _, k := range flipped.Keys {
		if (k.Provider == "openai") != k.IsDefault {
			t.Fatalf("is_default flags wrong after flip: %+v", flipped)
		}
	}

	// Delete the default (openai) → anthropic is promoted back to default.
	h.do(t, http.MethodDelete, "/keys?provider=openai", nil, nil).Body.Close()
	resp = h.do(t, http.MethodGet, "/keys", nil, nil)
	var after keysResponse
	decode(t, resp, &after)
	if len(after.Keys) != 1 || after.DefaultProvider != "anthropic" || !after.Keys[0].IsDefault {
		t.Fatalf("delete-default did not promote survivor: %+v", after)
	}
}

func TestSetDefaultUnknownProvider404(t *testing.T) {
	h := newHarness(t)
	h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "anthropic", "key": "sk-ant-abcdefghij-4a2f"}, nil).Body.Close()
	// openai isn't connected → can't be made default.
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "default": true}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d, want 404", resp.StatusCode)
	}
}

// --- chat ---

// sseResult is the parsed outcome of a chat SSE stream.
type sseResult struct {
	text      string
	done      bool
	truncated bool
	errCode   string
	errMsg    string
}

func readSSE(t *testing.T, resp *http.Response) sseResult {
	t.Helper()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var res sseResult
	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(line[len("data:"):])
		var frame struct {
			Delta     string `json:"delta"`
			Done      bool   `json:"done"`
			Truncated bool   `json:"truncated"`
			Error     string `json:"error"`
			Message   string `json:"message"`
		}
		if err := json.Unmarshal([]byte(data), &frame); err != nil {
			continue
		}
		res.text += frame.Delta
		if frame.Done {
			res.done = true
			res.truncated = frame.Truncated
		}
		if frame.Error != "" {
			res.errCode = frame.Error
			res.errMsg = frame.Message
		}
	}
	return res
}

func (h *harness) storeOpenAIKey(t *testing.T, raw string) {
	t.Helper()
	h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "key": raw, "default_model": "gpt-4o-mini"}, nil).Body.Close()
}

func TestChatHappyPathStreamsAndPersists(t *testing.T) {
	h := newHarness(t)
	raw := "sk-openai-secret-value-1234"
	h.storeOpenAIKey(t, raw)

	resp := h.do(t, http.MethodPost, "/chat", map[string]any{
		"context": "concept:sliding-window", "kind": "concept", "label": "Concept — Sliding Window", "message": "explain it",
	}, map[string]string{headerCoachMode: ModeGeneral})
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("chat status %d: %s", resp.StatusCode, b)
	}
	res := readSSE(t, resp)
	if res.text != "Hi there" || !res.done {
		t.Fatalf("sse result = %+v, want text 'Hi there' done", res)
	}
	// The provider must have received the DECRYPTED key.
	if h.lastAuth != raw {
		t.Fatalf("provider saw auth %q, want %q", h.lastAuth, raw)
	}
	// The thread now holds the user turn + the assistant reply.
	msgs := h.store.messagesFor(h.account, "concept:sliding-window")
	if len(msgs) != 2 || msgs[0].Role != store.RoleUser || msgs[1].Role != store.RoleAssistant || msgs[1].Content != "Hi there" {
		t.Fatalf("persisted = %+v", msgs)
	}
}

func TestChatNoKeyReturns409(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409", resp.StatusCode)
	}
	var env struct {
		Error struct{ Code string } `json:"error"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&env)
	if env.Error.Code != "no_key" {
		t.Fatalf("code = %q, want no_key", env.Error.Code)
	}
}

func TestChatDisabledKeyReturns409(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-abcdefghij-cdef")
	_ = h.store.SetKeyEnabled(context.Background(), h.account, "openai", false)
	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status %d, want 409", resp.StatusCode)
	}
}

func TestChatProviderAuthFailureDisablesKey(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-bad-key-1234")
	h.provHandler = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}
	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	// No delta was streamed, so this is a clean 409 the client maps to Settings.
	if resp.StatusCode != http.StatusConflict {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		t.Fatalf("status %d, want 409: %s", resp.StatusCode, b)
	}
	resp.Body.Close()
	// The bad key is now disabled.
	kc, _ := h.store.GetKey(context.Background(), h.account, "openai")
	if kc.Enabled {
		t.Fatal("provider-auth failure did not disable the key")
	}
}

func (h *harness) storeAnthropicKey(t *testing.T, raw string) {
	t.Helper()
	h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "anthropic", "key": raw, "default_model": "claude-sonnet-5"}, nil).Body.Close()
}

// TestChatProviderLimitedKeepsKeyEnabled: an out-of-credit / billing / spend- or
// rate-limited provider account is NOT a bad key. The learner gets the friendly
// provider_limited message and the key stays enabled.
func TestChatProviderLimitedKeepsKeyEnabled(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider string
		status   int
		body     string
	}{
		{"openai insufficient_quota", "openai", 429, `{"error":{"message":"You exceeded your current quota","type":"insufficient_quota","code":"insufficient_quota"}}`},
		{"openai rate limit", "openai", 429, `{"error":{"message":"Rate limit reached","type":"requests","code":"rate_limit_exceeded"}}`},
		{"openai billing hard limit", "openai", 400, `{"error":{"message":"Billing hard limit has been reached","type":"invalid_request_error","code":"billing_hard_limit_reached"}}`},
		{"openai region", "openai", 403, `{"error":{"message":"Country, region, or territory not supported","type":"request_forbidden","code":"unsupported_country_region_territory"}}`},
		{"anthropic billing_error", "anthropic", 402, `{"type":"error","error":{"type":"billing_error","message":"There's an issue with your billing or payment information."}}`},
		{"anthropic spend limit", "anthropic", 400, `{"type":"error","error":{"type":"invalid_request_error","message":"You have reached your specified API usage limits. You will regain access on 2026-10-01 at 00:00 UTC."}}`},
		{"anthropic spend cap", "anthropic", 429, `{"type":"error","error":{"type":"rate_limit_error","message":"You have reached your API usage limits","details":{"error_code":"enforced_spend_limit_reached"}}}`},
		{"anthropic permission", "anthropic", 403, `{"type":"error","error":{"type":"permission_error","message":"no access"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			if tc.provider == "anthropic" {
				h.storeAnthropicKey(t, "sk-ant-good-key-1234")
			} else {
				h.storeOpenAIKey(t, "sk-openai-good-key-1234")
			}
			h.provHandler = func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}
			resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
			var env struct {
				Error struct{ Code, Message string } `json:"error"`
			}
			status := resp.StatusCode
			decode(t, resp, &env)
			if status != http.StatusTooManyRequests || env.Error.Code != "provider_limited" {
				t.Fatalf("status %d code %q, want 429 provider_limited", status, env.Error.Code)
			}
			if !strings.Contains(env.Error.Message, "out of credit or limited") {
				t.Fatalf("message = %q, want the friendly top-up copy", env.Error.Message)
			}
			kc, _ := h.store.GetKey(context.Background(), h.account, tc.provider)
			if !kc.Enabled {
				t.Fatal("a limited provider account disabled the key")
			}
		})
	}
}

// TestChatProviderLimitedMidStream: a quota error frame after some text surfaces as an
// SSE provider_limited error, keeps the key, and persists the partial reply.
func TestChatProviderLimitedMidStream(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-good-key-1234")
	h.provHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`data: {"choices":[{"delta":{"content":"Partial"}}]}` + "\n\n"))
		_, _ = w.Write([]byte(`data: {"error":{"message":"quota","type":"insufficient_quota","code":"insufficient_quota"}}` + "\n\n"))
	}
	res := readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil))
	if res.errCode != "provider_limited" || !strings.Contains(res.errMsg, "top up") || res.text != "Partial" {
		t.Fatalf("sse = %+v, want provider_limited after 'Partial'", res)
	}
	if kc, _ := h.store.GetKey(context.Background(), h.account, "openai"); !kc.Enabled {
		t.Fatal("a mid-stream quota error disabled the key")
	}
	msgs := h.store.messagesFor(h.account, "dashboard")
	if len(msgs) != 2 || msgs[1].Content != "Partial" {
		t.Fatalf("persisted = %+v, want the partial reply", msgs)
	}
}

// TestChatTruncatedReplyIsMarked: a reply the provider cut off at max_tokens gets the
// truncation note, live and persisted, and the done frame says truncated.
func TestChatTruncatedReplyIsMarked(t *testing.T) {
	h := newHarness(t)
	h.storeAnthropicKey(t, "sk-ant-good-key-1234")
	h.provHandler = func(w http.ResponseWriter, r *http.Request) { anthropicFrames(w, "max_tokens", true) }

	res := readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil))
	want := "Hi there\n\n" + truncationNote
	if res.text != want || !res.done || !res.truncated {
		t.Fatalf("sse = %+v, want %q done+truncated", res, want)
	}
	msgs := h.store.messagesFor(h.account, "dashboard")
	if len(msgs) != 2 || msgs[1].Content != want {
		t.Fatalf("persisted = %+v, want the marked reply", msgs)
	}

	// A complete reply is not marked.
	h.provHandler = func(w http.ResponseWriter, r *http.Request) { anthropicFrames(w, "end_turn", true) }
	res = readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "again"}, nil))
	if res.text != "Hi there" || res.truncated {
		t.Fatalf("sse = %+v, want an unmarked complete reply", res)
	}
}

// TestChatModeGateFromHeader confirms the SERVER-AUTHORITATIVE mode header drives the
// system prompt: the fake provider echoes the system message back, so we can assert that
// an attempt on an unsolved problem is Socratic/spoiler-free and a review is not.
func TestChatModeGateFromHeader(t *testing.T) {
	h := newHarness(t)
	h.storeOpenAIKey(t, "sk-openai-abcdefghij-cdef")
	h.provHandler = echoSystemOpenAI

	// attempt mode → spoiler-free system prompt.
	resp := h.do(t, http.MethodPost, "/chat", map[string]any{
		"context": "problem:16", "kind": "problem", "problemId": "16", "message": "just give me the answer",
	}, map[string]string{headerCoachMode: ModeAttempt})
	res := readSSE(t, resp)
	if !strings.Contains(res.text, "ACTIVE ATTEMPT") || !strings.Contains(res.text, "spoiler-free") {
		t.Fatalf("attempt prompt missing spoiler gate: %q", res.text)
	}

	// review mode → reviewer prompt.
	resp = h.do(t, http.MethodPost, "/chat", map[string]any{
		"context": "problem:16", "kind": "problem", "problemId": "16", "message": "review my code",
	}, map[string]string{headerCoachMode: ModeReview})
	res = readSSE(t, resp)
	if !strings.Contains(res.text, "POST-SOLVE REVIEW") {
		t.Fatalf("review prompt missing reviewer mode: %q", res.text)
	}

	// A client that omits the mode on a problem context defaults to the SAFE attempt mode.
	resp = h.do(t, http.MethodPost, "/chat", map[string]any{
		"context": "problem:16", "kind": "problem", "problemId": "16", "message": "answer please",
	}, nil)
	res = readSSE(t, resp)
	if !strings.Contains(res.text, "ACTIVE ATTEMPT") {
		t.Fatalf("missing-mode problem did not default to spoiler-free attempt: %q", res.text)
	}
}

// echoSystemOpenAI streams back the request's system message as the reply (test double).
func echoSystemOpenAI(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	system := ""
	for _, m := range body.Messages {
		if m.Role == "system" {
			system = m.Content
		}
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	fl, _ := w.(http.Flusher)
	payload, _ := json.Marshal(map[string]any{"choices": []map[string]any{{"delta": map[string]string{"content": system}}}})
	_, _ = w.Write([]byte("data: " + string(payload) + "\n\n"))
	if fl != nil {
		fl.Flush()
	}
	_, _ = w.Write([]byte("data: [DONE]\n\n"))
}

func TestChatRequiresJWT(t *testing.T) {
	h := newHarness(t)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, h.server.URL+"/chat", strings.NewReader(`{"context":"x","message":"y"}`))
	// no Authorization header
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d, want 401", resp.StatusCode)
	}
}
