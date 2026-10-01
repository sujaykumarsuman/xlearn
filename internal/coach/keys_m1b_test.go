package coach

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// m1-10 (M1b) handler behaviour: the model catalog endpoint, per-feature defaults, the
// AD-bound sealed pair, OpenAI's store:false, and usage capture.

// --- GET /models ---

func TestGetModelsServesTheDatedCatalog(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodGet, "/models", nil, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got struct {
		AsOf      string            `json:"as_of"`
		Providers []string          `json:"providers"`
		Defaults  map[string]string `json:"defaults"`
		Models    []struct {
			ID           string   `json:"id"`
			Provider     string   `json:"provider"`
			Label        string   `json:"label"`
			Capabilities []string `json:"capabilities"`
			AsOf         string   `json:"as_of"`
			Recommended  bool     `json:"recommended"`
			CoveredModel bool     `json:"covered_model"`
			Price        struct {
				In  int64 `json:"input_micros_per_mtok"`
				Out int64 `json:"output_micros_per_mtok"`
			} `json:"price"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if got.AsOf != CatalogAsOf {
		t.Fatalf("as_of = %q, want %q", got.AsOf, CatalogAsOf)
	}
	// Anthropic first, then OpenAI — the order the UI offers them in.
	if len(got.Providers) != 2 || got.Providers[0] != store.ProviderAnthropic || got.Providers[1] != store.ProviderOpenAI {
		t.Fatalf("providers = %v", got.Providers)
	}
	if got.Defaults[store.ProviderAnthropic] != AnthropicDefaultModel ||
		got.Defaults[store.ProviderOpenAI] != OpenAIDefaultModel {
		t.Fatalf("defaults = %v", got.Defaults)
	}

	byID := map[string]bool{}
	for _, m := range got.Models {
		byID[m.ID] = true
		if m.AsOf != CatalogAsOf || m.Label == "" || m.Price.In <= 0 || m.Price.Out <= 0 {
			t.Fatalf("%s served incompletely: %+v", m.ID, m)
		}
		if len(m.Capabilities) == 0 {
			t.Fatalf("%s served no capabilities (the UI renders them as tags)", m.ID)
		}
	}
	// The ids this sprint adds must actually be served, not just present in Go.
	for _, id := range []string{"claude-opus-5-5", "gpt-6-sol", "gpt-6-luna", "gpt-6-astra"} {
		if !byID[id] {
			t.Errorf("GET /models does not serve %q", id)
		}
	}
}

// --- PUT /keys: model validation ---

func TestPutKeyRejectsMalformedModelIDs(t *testing.T) {
	h := newHarness(t)
	for _, tc := range []struct {
		name  string
		model string
	}{
		{"a URL", "https://proxy.example.com/v1"},
		{"a bare host and path", "proxy.example.com/v1"},
		{"spaces", "gpt 6 sol"},
		{"a leading slash", "/v1/chat"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := h.do(t, http.MethodPut, "/keys", map[string]any{
				"provider": "openai", "key": "sk-live-abcdefghijklmnop", "default_model": tc.model,
			}, nil)
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422", resp.StatusCode)
			}
			if code := errorCode(t, resp); code != "unknown_model" {
				t.Fatalf("error code = %q, want unknown_model", code)
			}
		})
	}
}

// TestPutKeyRejectsBaseURL is the SSRF guard: there is no base-URL field, and strict
// decoding means offering one is a 400 rather than a silently ignored field. Coach calls
// the endpoints this build was configured with, never one from a request body.
func TestPutKeyRejectsBaseURL(t *testing.T) {
	h := newHarness(t)
	for _, body := range []map[string]any{
		{"provider": "openai", "key": "sk-live-abcdefghijklmnop", "base_url": "http://169.254.169.254/"},
		{"provider": "openai", "key": "sk-live-abcdefghijklmnop", "baseUrl": "http://evil.example"},
		{"provider": "openai", "key": "sk-live-abcdefghijklmnop", "endpoint": "http://evil.example"},
	} {
		resp := h.do(t, http.MethodPut, "/keys", body, nil)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("body %v: status = %d, want 400", body, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

func TestPutKeyAcceptsACustomModelID(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "openai", "key": "sk-live-abcdefghijklmnop", "default_model": "ft:gpt-6-luna:personal:coach",
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	k, err := h.store.GetKey(t.Context(), h.account, store.ProviderOpenAI)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if k.DefaultModel != "ft:gpt-6-luna:personal:coach" {
		t.Fatalf("stored model = %q", k.DefaultModel)
	}
}

func TestPutKeyDefaultsToTheCatalogModel(t *testing.T) {
	h := newHarness(t)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "key": "sk-ant-abcdefghijklmnop",
	}, nil)
	resp.Body.Close()
	k, err := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if k.DefaultModel != AnthropicDefaultModel {
		t.Fatalf("model = %q, want the catalog default %q", k.DefaultModel, AnthropicDefaultModel)
	}
}

// --- PUT /keys: per-feature defaults ---

func TestInterviewDefaultRequiresAnInterviewCapableModel(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-abcdefghijklmnop")

	// A KNOWN catalog model without interview_brain is refused by name: we know this
	// model and know it is the wrong tool.
	noBrain := firstModelWithout(t, CapInterviewBrain)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "interview", "default_model": noBrain,
	}, nil)
	if resp.StatusCode != http.StatusUnprocessableEntity {
		resp.Body.Close()
		t.Fatalf("known non-interview model: status = %d, want 422", resp.StatusCode)
	}
	if code := errorCode(t, resp); code != "model_not_interview_capable" {
		t.Fatalf("error code = %q, want model_not_interview_capable", code)
	}
	resp.Body.Close()

	// A malformed id is `unknown_model`, not `model_not_interview_capable` — the learner
	// typed nonsense rather than picked the wrong model.
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "interview", "default_model": "https://evil.example/v1",
	}, nil)
	if code := errorCode(t, resp); code != "unknown_model" {
		t.Fatalf("malformed id error code = %q, want unknown_model", code)
	}
	resp.Body.Close()

	// A catalog model WITH interview_brain is accepted.
	brain := firstModelWith(t, CapInterviewBrain)
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "interview", "default_model": brain,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("interview-capable model: status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	if iv, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureInterview); err != nil || iv.FeatureModel != brain {
		t.Fatalf("interview default = (%q, %v), want %q", iv.FeatureModel, err, brain)
	}

	// A CUSTOM id is accepted too: it may be a fine-tune built for exactly this, it is
	// the learner's key, and m6a-02 derives "custom" from "not in the catalog".
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "interview", "default_model": "ft:custom:interviewer",
	}, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("custom interview id: status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()
	iv, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureInterview)
	if err != nil || iv.FeatureModel != "ft:custom:interviewer" {
		t.Fatalf("custom interview default = (%q, %v)", iv.FeatureModel, err)
	}
	// "Custom" must be derivable straight from the stored model id (m6a-02).
	if _, res := h.svcCatalog().ResolveModel(iv.FeatureModel); res != ModelCustom {
		t.Fatalf("stored interview model does not resolve as custom: %v", res)
	}
}

func TestPutKeyRejectsAnUnknownFeature(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-abcdefghijklmnop")
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "arena",
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", resp.StatusCode)
	}
	if code := errorCode(t, resp); code != "invalid_feature" {
		t.Fatalf("error code = %q, want invalid_feature", code)
	}
}

// TestSettingTheCoachDefaultLeavesInterviewAlone: the two features are independent, so a
// coach-side switch must not silently repoint the interview brain (which might then be a
// model that cannot run an interview).
func TestSettingTheCoachDefaultLeavesInterviewAlone(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-abcdefghijklmnop")
	h.connect(t, "openai", "sk-live-abcdefghijklmnop")

	brain := firstModelWith(t, CapInterviewBrain)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "interview", "default_model": brain,
	}, nil)
	resp.Body.Close()

	// Move the COACH default to the other provider.
	resp = h.do(t, http.MethodPut, "/keys", map[string]any{"provider": "openai", "default": true}, nil)
	resp.Body.Close()

	cd, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureCoach)
	if err != nil || cd.Provider != store.ProviderOpenAI {
		t.Fatalf("coach default = (%q, %v), want openai", cd.Provider, err)
	}
	iv, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureInterview)
	if err != nil || iv.Provider != store.ProviderAnthropic || iv.FeatureModel != brain {
		t.Fatalf("interview default moved: (%q, %q, %v)", iv.Provider, iv.FeatureModel, err)
	}
}

// --- GET /keys: defaults + usage_month ---

func TestGetKeysCarriesDefaultsAndUsage(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-abcdefghijklmnop")

	resp := h.do(t, http.MethodGet, "/keys", nil, nil)
	defer resp.Body.Close()
	var got keysResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// The v1 shape is preserved for an open v1.6.0 tab …
	if !got.Connected || got.DefaultProvider != store.ProviderAnthropic {
		t.Fatalf("v1 fields: connected=%v default_provider=%q", got.Connected, got.DefaultProvider)
	}
	if len(got.Keys) != 1 || !got.Keys[0].IsDefault {
		t.Fatalf("keys = %+v", got.Keys)
	}
	// … and both features are PRESENT as keys, with interview null (the normal starting
	// state, rendered "Not set"). Omitting the key would be indistinguishable from a
	// release that doesn't know the feature.
	if _, ok := got.Defaults[store.FeatureCoach]; !ok {
		t.Fatal("defaults has no coach key")
	}
	if iv, ok := got.Defaults[store.FeatureInterview]; !ok || iv != nil {
		t.Fatalf("defaults[interview] = %v, want present and null", iv)
	}
	if d := got.Defaults[store.FeatureCoach]; d == nil || d.Provider != store.ProviderAnthropic || d.Model != AnthropicDefaultModel {
		t.Fatalf("defaults[coach] = %+v", d)
	}
	// No chats yet: a zeroed, known-cost usage block (not absent, not unknown).
	if got.UsageMonth.Messages != 0 || got.UsageMonth.HasUnknownCost {
		t.Fatalf("usage_month = %+v, want zeroed with has_unknown_cost=false", got.UsageMonth)
	}
}

// --- store:false on every OpenAI request ---

// TestOpenAIAlwaysSendsStoreFalse is the retention guard. v1 left `store` at OpenAI's
// default (true), which logged every learner's coach conversation into their own OpenAI
// org — study conversations about problems they are being assessed on. The assertion is
// on EVERY request the provider makes, not just a happy-path chat, so a future code path
// through Stream cannot quietly omit it.
func TestOpenAIAlwaysSendsStoreFalse(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		bodies = append(bodies, body)
		openAISSE(w)
	}))
	defer srv.Close()

	p := NewOpenAIProvider(srv.URL, srv.Client())
	// Several shapes: a reasoning model, a non-reasoning model, an empty system prompt,
	// a long history.
	for _, model := range []string{"gpt-5.6-sol", "gpt-4o", "gpt-6-sol", "ft:gpt-6-luna:org:x"} {
		if _, _, err := collectModel(t, p, "sk", model); err != nil {
			t.Fatalf("stream %s: %v", model, err)
		}
	}
	if len(bodies) == 0 {
		t.Fatal("the fake provider saw no requests")
	}
	for i, body := range bodies {
		store, ok := body["store"]
		if !ok {
			t.Fatalf("request %d omitted `store` entirely", i)
		}
		if store != false {
			t.Fatalf("request %d sent store=%v, want false", i, store)
		}
		so, ok := body["stream_options"].(map[string]any)
		if !ok || so["include_usage"] != true {
			t.Fatalf("request %d stream_options = %v, want include_usage:true", i, body["stream_options"])
		}
	}
}

// TestOpenAIParsesTheUsageFrame: the include_usage chunk arrives with EMPTY choices, so a
// parser that only looks at choices would drop it and leave every turn uncosted.
func TestOpenAIParsesTheUsageFrame(t *testing.T) {
	srv := sseServer(
		`data: {"choices":[{"delta":{"content":"Hi"}}]}`,
		`data: {"choices":[{"delta":{},"finish_reason":"stop"}]}`,
		`data: {"choices":[],"usage":{"prompt_tokens":1234,"completion_tokens":567,"total_tokens":1801}}`,
		`data: [DONE]`,
	)
	defer srv.Close()
	_, res, err := collect(t, NewOpenAIProvider(srv.URL, srv.Client()), "sk")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if !res.Usage.HasTokens || res.Usage.InputTokens != 1234 || res.Usage.OutputTokens != 567 {
		t.Fatalf("usage = %+v, want 1234/567", res.Usage)
	}
}

// TestOpenAIWithoutUsageFrameReportsNoTokens: "the provider told us nothing" must stay
// distinguishable from "zero tokens", or a missing usage frame would be recorded as a
// free turn and under-report the month.
func TestOpenAIWithoutUsageFrameReportsNoTokens(t *testing.T) {
	srv := sseServer(`data: {"choices":[{"delta":{"content":"Hi"}}]}`, `data: [DONE]`)
	defer srv.Close()
	_, res, err := collect(t, NewOpenAIProvider(srv.URL, srv.Client()), "sk")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if res.Usage.HasTokens {
		t.Fatalf("usage = %+v, want HasTokens=false", res.Usage)
	}
}

func TestAnthropicParsesUsageAcrossFrames(t *testing.T) {
	// input_tokens arrives on message_start, output_tokens is final on message_delta.
	srv := sseServer(
		`event: message_start`+"\n"+`data: {"type":"message_start","message":{"usage":{"input_tokens":900,"output_tokens":1}}}`,
		`event: content_block_delta`+"\n"+`data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`,
		`event: message_delta`+"\n"+`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}}`,
		`event: message_stop`+"\n"+`data: {"type":"message_stop"}`,
	)
	defer srv.Close()
	_, res, err := collect(t, NewAnthropicProvider(srv.URL, srv.Client()), "sk")
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if !res.Usage.HasTokens || res.Usage.InputTokens != 900 || res.Usage.OutputTokens != 42 {
		t.Fatalf("usage = %+v, want 900/42", res.Usage)
	}
	if res.StopReason != "end_turn" {
		t.Fatalf("stop reason = %q", res.StopReason)
	}
}

// --- the AD-bound pair ---

// TestPutKeyWritesBothSealedPairs: a PUT must write the legacy pair (so v1.6.0 can still
// read the key during a rollback) AND the AD pair with its kek_id and digest, in one go.
// If only one were written they could diverge, which is the whole failure mode
// ad_src_digest exists to detect.
func TestPutKeyWritesBothSealedPairs(t *testing.T) {
	h := newHarness(t)
	const raw = "sk-ant-this-is-the-secret-1234"
	h.connect(t, "anthropic", raw)

	k, err := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if len(k.EncKey) == 0 || len(k.EncDataKey) == 0 {
		t.Fatal("the legacy pair was not written; a rollback to v1.6.0 could not read this key")
	}
	if len(k.EncKeyAD) == 0 || len(k.EncDataKeyAD) == 0 {
		t.Fatal("the AD pair was not written")
	}
	if k.KEKID != DefaultKEKID {
		t.Fatalf("kek_id = %q, want %q", k.KEKID, DefaultKEKID)
	}
	if len(k.ADSrcDigest) != 32 {
		t.Fatalf("ad_src_digest is %d bytes, want 32", len(k.ADSrcDigest))
	}
	if !k.ADPairCurrent() {
		t.Fatal("a freshly written row must have a current AD pair")
	}

	// Both pairs decrypt to the same secret, each by its own route.
	legacy, err := h.cipher.Open(k.EncKey, k.EncDataKey)
	if err != nil || string(legacy) != raw {
		t.Fatalf("legacy open = (%q, %v)", legacy, err)
	}
	ad := secrets.CoachKeyAD(k.AccountID, k.Provider)
	bound, err := h.keys.OpenAD(k.EncKeyAD, k.EncDataKeyAD, k.KEKID, ad)
	if err != nil || string(bound) != raw {
		t.Fatalf("ad open = (%q, %v)", bound, err)
	}
	// And the AD pair really is bound: another account's ad must not open it.
	if _, err := h.keys.OpenAD(k.EncKeyAD, k.EncDataKeyAD, k.KEKID,
		secrets.CoachKeyAD("22222222-2222-4222-8222-222222222222", k.Provider)); err == nil {
		t.Fatal("the AD pair opened under another account's associated data")
	}
}

// TestChatPrefersTheADPair / TestChatFallsBackForAStalePair pin the read rule. The stale
// case is the one that matters operationally: a v1.6.0 image replacing a key during a
// rollback rewrites ONLY the legacy columns, so the AD pair still holds the OLD key. Chat
// must use the newer legacy pair — otherwise, if the owner rotated because the old key was
// revoked, the 401 would disable the key they just pasted.
func TestChatPrefersTheADPair(t *testing.T) {
	h := newHarness(t)
	const raw = "sk-ant-ad-pair-key-9999"
	h.connect(t, "anthropic", raw)
	h.provHandler = func(w http.ResponseWriter, r *http.Request) { anthropicSSE(w) }

	// Corrupt the LEGACY pair only. A current AD pair must still answer, proving chat
	// read the AD pair and not the legacy one.
	k, _ := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	k.EncKey = []byte("corrupt-legacy-bytes-corrupt-legacy")
	h.replaceStoredKey(t, k)

	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("status = %d, want 200 (the AD pair should have answered)", resp.StatusCode)
	}
	// Drain before reading lastAuth: the fake provider writes it from the test server's
	// goroutine, so the stream has to have finished first.
	readSSE(t, resp)
	if h.lastAuth != raw {
		t.Fatalf("provider saw key %q, want the AD-pair key", h.lastAuth)
	}
}

func TestChatFallsBackForAStalePair(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-old-key-0000")
	h.provHandler = func(w http.ResponseWriter, r *http.Request) { anthropicSSE(w) }

	// Simulate a v1.6.0 key replace: rewrite ONLY the legacy pair, leaving the AD pair
	// and its digest describing the previous key.
	const newRaw = "sk-ant-new-key-1111"
	k, _ := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	encKey, encDataKey, err := h.cipher.Seal([]byte(newRaw))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	k.EncKey, k.EncDataKey = encKey, encDataKey // AD pair + digest untouched → now STALE
	h.replaceStoredKey(t, k)

	stale, _ := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	if stale.ADPairCurrent() {
		t.Fatal("the AD pair should read as stale after a legacy-only rewrite")
	}

	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	readSSE(t, resp) // drain before reading lastAuth
	if h.lastAuth != newRaw {
		t.Fatalf("provider saw %q, want the NEW key %q — a stale AD pair was used", h.lastAuth, newRaw)
	}
}

// TestChatDoesNotFallBackFromAFailingCurrentPair: if a CURRENT AD pair fails to open, that
// is final. Falling back would defeat the binding entirely — anyone able to write the
// database could corrupt the AD pair to be served the unbound legacy one instead.
func TestChatDoesNotFallBackFromAFailingCurrentPair(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-key-2222")

	k, _ := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	// Tamper with the AD ciphertext but keep the digest matching the legacy pair, so the
	// pair still reads as CURRENT.
	k.EncKeyAD = append([]byte(nil), k.EncKeyAD...)
	k.EncKeyAD[len(k.EncKeyAD)-1] ^= 0xff
	h.replaceStoredKey(t, k)

	cur, _ := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
	if !cur.ADPairCurrent() {
		t.Fatal("the tampered pair must still read as current for this test to mean anything")
	}

	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 (no fallback from a current-but-failing AD pair)", resp.StatusCode)
	}
	if h.lastAuth != "" {
		t.Fatalf("the provider was called with %q after a failed decrypt", h.lastAuth)
	}
}

// TestADPairCurrentRule covers the predicate directly, including the arm that cannot yet
// be produced in the database: from l-01 on, coach stops writing the legacy pair, and a
// row with a NULL legacy pair must read its AD pair rather than be treated as stale
// forever. (api_key_config.enc_key / enc_data_key are still NOT NULL until l-01 does the
// DROP NOT NULL, so this arm is unit-tested rather than exercised through Postgres.)
func TestADPairCurrentRule(t *testing.T) {
	digestOf := func(b []byte) []byte {
		d := sha256.Sum256(b)
		return d[:]
	}
	legacy := []byte("legacy-wrapped-data-key")
	for _, tc := range []struct {
		name string
		k    store.KeyConfig
		want bool
	}{
		{"no AD pair at all", store.KeyConfig{EncKey: []byte("x"), EncDataKey: legacy}, false},
		{"AD key but no AD data key", store.KeyConfig{EncKeyAD: []byte("a"), EncDataKey: legacy}, false},
		{"digest matches the legacy pair", store.KeyConfig{
			EncKeyAD: []byte("a"), EncDataKeyAD: []byte("b"), EncDataKey: legacy, ADSrcDigest: digestOf(legacy),
		}, true},
		{"digest is stale", store.KeyConfig{
			EncKeyAD: []byte("a"), EncDataKeyAD: []byte("b"), EncDataKey: []byte("rewritten"), ADSrcDigest: digestOf(legacy),
		}, false},
		{"digest missing (a pre-00006 row)", store.KeyConfig{
			EncKeyAD: []byte("a"), EncDataKeyAD: []byte("b"), EncDataKey: legacy,
		}, false},
		{"legacy pair gone (l-01 onwards)", store.KeyConfig{
			EncKeyAD: []byte("a"), EncDataKeyAD: []byte("b"),
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.k.ADPairCurrent(); got != tc.want {
				t.Fatalf("ADPairCurrent() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- usage capture on the assistant turn ---

func TestChatStoresUsageAndCost(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-usage-key-3333")
	// Pin the model so the expected cost is the catalog's published price for it.
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default_model": "claude-sonnet-5",
	}, nil)
	resp.Body.Close()

	h.provHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, f := range []string{
			`event: message_start` + "\n" + `data: {"type":"message_start","message":{"usage":{"input_tokens":3000,"output_tokens":0}}}`,
			`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`,
			`event: message_delta` + "\n" + `data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":700}}`,
			`event: message_stop` + "\n" + `data: {"type":"message_stop"}`,
		} {
			_, _ = w.Write([]byte(f + "\n\n"))
		}
	}

	// Drain the stream to completion before asserting: the assistant turn is persisted
	// after the last SSE frame is written, so closing the body early would race the
	// handler.
	readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil))

	us := h.store.usageFor(h.account, threadKeyDashboard)
	if len(us) != 1 {
		t.Fatalf("recorded %d assistant turns, want 1", len(us))
	}
	u := us[0]
	if u.Provider != store.ProviderAnthropic || u.Model != "claude-sonnet-5" {
		t.Fatalf("usage provider/model = %q/%q", u.Provider, u.Model)
	}
	if !u.HasTokens || u.InputTokens != 3000 || u.OutputTokens != 700 {
		t.Fatalf("usage tokens = %+v, want 3000/700", u)
	}
	if u.StopReason != "end_turn" {
		t.Fatalf("stop reason = %q", u.StopReason)
	}
	// $2/MTok in + $10/MTok out → 3000*2 + 700*10 = 6000 + 7000 = 13_000 micros.
	if u.EstCostMicros == nil || *u.EstCostMicros != 13_000 {
		t.Fatalf("est_cost_micros = %v, want 13000", deref(u.EstCostMicros))
	}

	// And it surfaces in GET /keys as usage_month.
	kr := h.keysResponse(t)
	if kr.UsageMonth.Messages != 1 || kr.UsageMonth.EstCostMicros != 13_000 || kr.UsageMonth.HasUnknownCost {
		t.Fatalf("usage_month = %+v", kr.UsageMonth)
	}
}

// TestChatStoresUnknownCostForACustomModel: a custom id has no published price, so the
// cost is recorded as UNKNOWN rather than as zero — which is what makes the month line say
// "custom models not estimated" instead of quietly under-reporting.
func TestChatStoresUnknownCostForACustomModel(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-custom-4444")
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default_model": "ft:claude:personal:coach",
	}, nil)
	resp.Body.Close()

	h.provHandler = func(w http.ResponseWriter, r *http.Request) { anthropicSSE(w) }
	// Drain before asserting (see TestChatStoresUsageAndCost).
	readSSE(t, h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil))

	us := h.store.usageFor(h.account, threadKeyDashboard)
	if len(us) != 1 {
		t.Fatalf("recorded %d turns, want 1", len(us))
	}
	if us[0].EstCostMicros != nil {
		t.Fatalf("est_cost_micros = %d for a custom model, want unknown (nil)", *us[0].EstCostMicros)
	}
	kr := h.keysResponse(t)
	if !kr.UsageMonth.HasUnknownCost {
		t.Fatalf("usage_month = %+v, want has_unknown_cost=true", kr.UsageMonth)
	}
}

// --- typed error reasons on the wire ---

// TestChatErrorCarriesTheReason pins the `reason` the SPA switches on to render AB01's
// copy, in both shapes: the pre-stream JSON envelope and the mid-stream SSE error frame.
// The v1 `error` code and `message` must survive alongside it so a v1.6.0 tab still works.
func TestChatErrorCarriesTheReason(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		wantCode   string
		wantReason string
		wantStatus int
		wantEnable bool // is the key still enabled afterwards?
	}{
		{
			name: "auth disables the key", status: 401,
			body:     `{"type":"error","error":{"type":"authentication_error","message":"bad key"}}`,
			wantCode: "provider_auth", wantReason: "auth", wantStatus: http.StatusConflict, wantEnable: false,
		},
		{
			name: "quota keeps the key", status: 402,
			body:     `{"type":"error","error":{"type":"billing_error","message":"out of credit"}}`,
			wantCode: "provider_limited", wantReason: "quota", wantStatus: http.StatusTooManyRequests, wantEnable: true,
		},
		{
			name: "rate limit keeps the key", status: 429,
			body:     `{"type":"error","error":{"type":"rate_limit_error","message":"slow down"}}`,
			wantCode: "provider_limited", wantReason: "rate_limit", wantStatus: http.StatusTooManyRequests, wantEnable: true,
		},
		{
			name: "model access keeps the key", status: 404,
			body:     `{"type":"error","error":{"type":"not_found_error","message":"model: claude-nope"}}`,
			wantCode: "provider_limited", wantReason: "model_access", wantStatus: http.StatusTooManyRequests, wantEnable: true,
		},
		{
			name: "region keeps the key", status: 403,
			body:     `{"type":"error","error":{"type":"permission_error","message":"not available in your region"}}`,
			wantCode: "provider_limited", wantReason: "region", wantStatus: http.StatusTooManyRequests, wantEnable: true,
		},
		{
			name: "upstream failure keeps the key", status: 500,
			body:     `{"type":"error","error":{"type":"api_error","message":"boom"}}`,
			wantCode: "provider_error", wantReason: "unavailable", wantStatus: http.StatusBadGateway, wantEnable: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)
			h.connect(t, "anthropic", "sk-ant-reason-5555")
			h.provHandler = func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}

			resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
			var env struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
					Reason  string `json:"reason"`
				} `json:"error"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if env.Error.Code != tc.wantCode {
				t.Fatalf("code = %q, want %q", env.Error.Code, tc.wantCode)
			}
			if env.Error.Reason != tc.wantReason {
				t.Fatalf("reason = %q, want %q", env.Error.Reason, tc.wantReason)
			}
			if env.Error.Message == "" {
				t.Fatal("the v1 `message` is gone; a v1.6.0 tab shows nothing")
			}

			// Only auth disables.
			k, err := h.store.GetKey(t.Context(), h.account, store.ProviderAnthropic)
			if err != nil {
				t.Fatalf("get key: %v", err)
			}
			if k.Enabled != tc.wantEnable {
				t.Fatalf("key enabled = %v, want %v (only auth may disable)", k.Enabled, tc.wantEnable)
			}
		})
	}
}

// TestMidStreamErrorCarriesTheReason: once bytes are on the wire the outcome can only be
// an SSE frame, which must carry the same three fields.
func TestMidStreamErrorCarriesTheReason(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-midstream-6666")
	h.provHandler = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`event: content_block_delta` + "\n" + `data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"partial"}}` + "\n\n"))
		_, _ = w.Write([]byte(`event: error` + "\n" + `data: {"type":"error","error":{"type":"not_found_error","message":"model gone"}}` + "\n\n"))
	}

	resp := h.do(t, http.MethodPost, "/chat", map[string]any{"context": "dashboard", "message": "hi"}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 (the stream had already started)", resp.StatusCode)
	}
	body := readAll(t, resp)
	if !strings.Contains(body, `"reason":"model_access"`) {
		t.Fatalf("SSE error frame has no model_access reason:\n%s", body)
	}
	if !strings.Contains(body, `"error":"provider_limited"`) {
		t.Fatalf("SSE error frame lost its v1 error code:\n%s", body)
	}
	// The partial reply is still persisted, so a reload doesn't leave a dangling user turn.
	msgs := h.store.messagesFor(h.account, threadKeyDashboard)
	if len(msgs) != 2 || msgs[1].Role != store.RoleAssistant || !strings.Contains(msgs[1].Content, "partial") {
		t.Fatalf("messages = %+v", msgs)
	}
}

// --- helpers ---

// connect stores a key for provider via the real PUT handler (so it goes through sealKey).
func (h *harness) connect(t *testing.T, provider, rawKey string) {
	t.Helper()
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{"provider": provider, "key": rawKey}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("connect %s: status %d", provider, resp.StatusCode)
	}
	h.lastAuth = ""
}

// replaceStoredKey writes k straight into the fake store, bypassing the handler — used to
// stage the byte-level states only another release (or a corrupted row) could produce.
func (h *harness) replaceStoredKey(t *testing.T, k store.KeyConfig) {
	t.Helper()
	h.store.mu.Lock()
	defer h.store.mu.Unlock()
	h.store.keys[k.AccountID][k.Provider] = k
}

// keysResponse decodes GET /keys.
func (h *harness) keysResponse(t *testing.T) keysResponse {
	t.Helper()
	resp := h.do(t, http.MethodGet, "/keys", nil, nil)
	defer resp.Body.Close()
	var out keysResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode keys: %v", err)
	}
	return out
}

// svcCatalog exposes the catalog a harness's service uses.
func (h *harness) svcCatalog() *Catalog { return NewCatalog() }

// threadKeyDashboard is the thread key the context "dashboard" normalizes to: a v1
// account-wide context belongs to the default course (course.NormalizeCoachContext,
// m1-03), so the stored thread is "dsa:dashboard" even though the client sent "dashboard".
const threadKeyDashboard = "dsa:dashboard"

// readAll drains a response body as a string.
func readAll(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

// errorCode reads the error envelope's code.
func errorCode(t *testing.T, resp *http.Response) string {
	t.Helper()
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	return env.Error.Code
}

// firstModelWith / firstModelWithout pick a catalog id by capability, so these tests keep
// working as the catalog changes rather than hard-coding a model that may be retired.
func firstModelWith(t *testing.T, capability string) string {
	t.Helper()
	for _, m := range NewCatalog().Models() {
		if m.HasCapability(capability) {
			return m.ID
		}
	}
	t.Fatalf("no catalog model has %q", capability)
	return ""
}

func firstModelWithout(t *testing.T, capability string) string {
	t.Helper()
	for _, m := range NewCatalog().Models() {
		if !m.HasCapability(capability) {
			return m.ID
		}
	}
	t.Fatalf("every catalog model has %q", capability)
	return ""
}
