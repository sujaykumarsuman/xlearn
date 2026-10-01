package coach

import (
	"errors"
	"net/http"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// TestInterviewDefaultValidatesTheDerivedModelToo closes a hole the m1-10 adversarial
// review found: the handler validated `default_model` only when the client SENT one, and
// the store then fell back to the key's own `default_model` — unvalidated.
//
// So `{"provider":"openai","default":true,"feature":"interview"}` with no model stored
// `key_default(interview, model="gpt-5.6-sol")`, a chat-only catalog entry — exactly the
// state the sprint's acceptance criterion says must be a 422
// `model_not_interview_capable`. It matters beyond tidiness: m6a-02 reads this row and
// derives "custom" from "absent from the catalog", so it cannot tell a deliberate custom
// brain from a silently-defaulted chat model. The current SPA always sends a model, so
// this was reachable only through the API — which is why nothing surfaced it.
func TestInterviewDefaultValidatesTheDerivedModelToo(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "openai", "sk-live-abcdefghijklmnop")

	// Pin the key's model to a known chat-only catalog entry rather than relying on the
	// provider default happening to be one. Today it is (OpenAIDefaultModel is
	// gpt-5.6-sol), but if a later release promotes an interview-capable model this test
	// would quietly stop testing anything — which is the moment it matters most.
	noBrain := firstModelWithout(t, CapInterviewBrain)
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "openai", "default_model": noBrain,
	}, nil)
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		t.Fatalf("seed the key's model: status %d", resp.StatusCode)
	}
	resp.Body.Close()

	k, err := h.store.GetKey(t.Context(), h.account, store.ProviderOpenAI)
	if err != nil {
		t.Fatalf("get key: %v", err)
	}
	if ok, _ := NewCatalog().InterviewCapable(k.DefaultModel); ok {
		t.Fatalf("setup: %q is interview-capable, so this test proves nothing", k.DefaultModel)
	}

	resp = h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "openai", "default": true, "feature": "interview", // no default_model
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422 — the derived model %q is chat-only", resp.StatusCode, k.DefaultModel)
	}
	if code := errorCode(t, resp); code != "model_not_interview_capable" {
		t.Fatalf("error code = %q, want model_not_interview_capable", code)
	}
	if _, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureInterview); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("an interview default was stored anyway (err = %v)", err)
	}
}

// TestInterviewDefaultDerivesAnInterviewCapableModel: the same omitted-model path must
// still SUCCEED when the key's own model happens to be interview-capable, and must store
// that model rather than leaving it empty.
func TestInterviewDefaultDerivesAnInterviewCapableModel(t *testing.T) {
	h := newHarness(t)
	brain := firstModelWith(t, CapInterviewBrain)
	h.connect(t, "anthropic", "sk-ant-abcdefghijklmnop")

	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default_model": brain,
	}, nil)
	resp.Body.Close()

	resp = h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, "feature": "interview",
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	iv, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureInterview)
	if err != nil || iv.FeatureModel != brain {
		t.Fatalf("interview default = (%q, %v), want %q", iv.FeatureModel, err, brain)
	}
}

// TestCoachDefaultStillTakesTheKeysModelWhenOmitted: the fix must not make `coach`
// stricter. Omitting the model there is the normal case (the header pill sends one; a bare
// set-default does not), and any model a key already carries is acceptable for chat —
// including a custom id the catalog has never heard of.
func TestCoachDefaultStillTakesTheKeysModelWhenOmitted(t *testing.T) {
	h := newHarness(t)
	h.connect(t, "anthropic", "sk-ant-abcdefghijklmnop")
	resp := h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default_model": "ft:claude:personal:coach",
	}, nil)
	resp.Body.Close()

	resp = h.do(t, http.MethodPut, "/keys", map[string]any{
		"provider": "anthropic", "default": true, // no feature, no model → coach
	}, nil)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	cd, err := h.store.GetDefaultKey(t.Context(), h.account, store.FeatureCoach)
	if err != nil || cd.FeatureModel != "ft:claude:personal:coach" {
		t.Fatalf("coach default = (%q, %v), want the key's custom model", cd.FeatureModel, err)
	}
}
