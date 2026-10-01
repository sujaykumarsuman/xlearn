package coach

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// TestCatalogIsWellFormed pins the invariants the UI and the cost arithmetic depend on.
func TestCatalogIsWellFormed(t *testing.T) {
	c := NewCatalog()

	if c.AsOf() == "" {
		t.Fatal("catalog as_of is empty; the UI renders prices with their date")
	}
	if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(c.AsOf()) {
		t.Fatalf("as_of = %q, want YYYY-MM-DD", c.AsOf())
	}

	seen := map[string]bool{}
	recommended := map[string]int{}
	for _, m := range c.Models() {
		switch {
		case m.ID == "":
			t.Fatal("a catalog entry has no id")
		case seen[m.ID]:
			t.Fatalf("duplicate catalog id %q", m.ID)
		case m.Label == "":
			t.Fatalf("%s has no label", m.ID)
		case !store.ValidProvider(m.Provider):
			t.Fatalf("%s names provider %q, which the DB CHECK does not allow", m.ID, m.Provider)
		case m.AsOf != c.AsOf():
			t.Fatalf("%s as_of = %q, catalog as_of = %q", m.ID, m.AsOf, c.AsOf())
		case !m.HasCapability(CapChat):
			t.Fatalf("%s is not chat-capable; every coach model must be", m.ID)
		}
		seen[m.ID] = true
		if m.Recommended {
			recommended[m.Provider]++
		}

		// A price of zero would quietly turn a real cost into "free" in the month
		// summary, which is worse than "unknown". Every catalog entry must be priced —
		// an unpriced model belongs outside the catalog, where it reads "cost unknown".
		if m.Price.InputMicrosPerMTok <= 0 || m.Price.OutputMicrosPerMTok <= 0 {
			t.Fatalf("%s has a non-positive price %+v", m.ID, m.Price)
		}

		// Capabilities are a closed set: the UI maps each to a tag, and a typo would
		// silently disappear from the switcher rather than fail.
		for _, cap := range m.Capabilities {
			switch cap {
			case CapChat, CapInterviewBrain, CapVoiceShell:
			default:
				t.Fatalf("%s carries unknown capability %q", m.ID, cap)
			}
		}
	}

	// Exactly one "Recommended" badge per provider, or the switcher shows two or none.
	for _, p := range []string{store.ProviderAnthropic, store.ProviderOpenAI} {
		if recommended[p] != 1 {
			t.Fatalf("provider %s has %d recommended models, want exactly 1", p, recommended[p])
		}
	}

	// Each provider's default must BE in the catalog and be the recommended entry — the
	// badge and the default a new key gets have to agree, or onboarding silently sets a
	// model the UI recommends against.
	for _, p := range []string{store.ProviderAnthropic, store.ProviderOpenAI} {
		id := c.DefaultModel(p)
		m, ok := c.Lookup(id)
		if !ok {
			t.Fatalf("provider %s defaults to %q, which is not in the catalog", p, id)
		}
		if m.Provider != p {
			t.Fatalf("provider %s defaults to %q, which belongs to %s", p, id, m.Provider)
		}
		if !m.Recommended {
			t.Fatalf("provider %s defaults to %q, which is not the recommended entry", p, id)
		}
	}
}

// TestCatalogCarriesTheM1bAdditions pins the four ids the sprint adds, plus the v1 entries
// a learner may already have set — dropping one of those would make a working model read
// as "custom, cost unknown" after an upgrade.
func TestCatalogCarriesTheM1bAdditions(t *testing.T) {
	c := NewCatalog()
	for _, id := range []string{
		// m1-10 additions.
		"claude-opus-5-5", "gpt-6-sol", "gpt-6-luna", "gpt-6-astra",
		// The v1 SPA's list, still served by the providers.
		"claude-opus-5", "claude-opus-4-8", "claude-sonnet-5",
		"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna",
	} {
		if _, ok := c.Lookup(id); !ok {
			t.Errorf("catalog is missing %q", id)
		}
	}
}

// TestNoCatalogEntryCarriesVoiceShell: no entry carries voice_shell in this release. S6
// HAS reported and the shell is settled (`gpt-live-1` per D42, with
// `gpt-realtime-2.1-mini`), so this is not "waiting for S6" — it is that a realtime shell
// is billed PER MINUTE, which the per-MTok Price here cannot state, and that everything in
// this catalog is offered as a selectable coach model. See the voice_shell note in
// catalog.go; spk-04's hand-off is re-pointed at m6b-01 in docs/v2/status.md.
func TestNoCatalogEntryCarriesVoiceShell(t *testing.T) {
	for _, m := range NewCatalog().Models() {
		if m.HasCapability(CapVoiceShell) {
			t.Errorf("%s carries voice_shell: give it a per-minute price shape (m6b-01's "+
				"billing_shape) and keep it out of the chat pickers before listing it", m.ID)
		}
	}
}

// TestOnlyChatModelsAreOffered is the guard that makes adding a voice shell later safe.
// GET /models feeds the coach and interview model pickers, so a non-chat entry served
// there is a model a learner can select and then have every chat fail on.
func TestOnlyChatModelsAreOffered(t *testing.T) {
	c := NewCatalog()

	// Today every entry is chat-capable, so the two lists match.
	if len(c.ChatModels()) != len(c.Models()) {
		t.Fatalf("ChatModels has %d of %d entries; update this test if a non-chat entry was added",
			len(c.ChatModels()), len(c.Models()))
	}
	for _, m := range c.ChatModels() {
		if !m.HasCapability(CapChat) {
			t.Fatalf("ChatModels returned %s, which is not chat-capable", m.ID)
		}
	}

	// And the filter really filters: a voice-only entry must not come back.
	voiceOnly := &Catalog{
		asOf: CatalogAsOf,
		models: []Model{
			{ID: "chatty", Provider: store.ProviderOpenAI, Capabilities: []string{CapChat}},
			{ID: "gpt-live-1", Provider: store.ProviderOpenAI, Capabilities: []string{CapVoiceShell}},
		},
		byID:     map[string]Model{},
		defaults: map[string]string{},
	}
	got := voiceOnly.ChatModels()
	if len(got) != 1 || got[0].ID != "chatty" {
		t.Fatalf("ChatModels = %+v, want only the chat-capable entry", got)
	}
}

// TestNoCatalogEntryIsACoveredModel documents why covered_model is false everywhere today:
// the Covered Models are Anthropic's Fable and Mythos families, which coach does not list.
// Covered Models are ALLOWED on a BYO key (it is the learner's own provider account) — the
// UI labels them rather than blocking them — so this test is a reminder to set the flag if
// one is ever added, not a ban.
func TestNoCatalogEntryIsACoveredModel(t *testing.T) {
	for _, m := range NewCatalog().Models() {
		if m.CoveredModel {
			t.Errorf("%s is flagged covered_model; confirm the 30-day retention label is shown", m.ID)
		}
	}
}

func TestResolveModel(t *testing.T) {
	c := NewCatalog()
	for _, tc := range []struct {
		name string
		id   string
		want ModelResolution
	}{
		{"catalog entry", "claude-sonnet-5", ModelKnown},
		{"catalog entry with trailing space", " claude-sonnet-5 ", ModelKnown},
		{"a fine-tune", "ft:gpt-6-luna:personal:coach", ModelCustom},
		{"a dated snapshot this release doesn't list", "claude-haiku-4-5-20251001", ModelCustom},
		{"a plain unknown id", "some-new-model-9", ModelCustom},
		{"64 chars", strings.Repeat("a", 64), ModelCustom},

		// Rejected: these are the shapes that would turn "pick a model" into "pick an
		// endpoint". A URL is the one a hostile client actually tries.
		{"empty", "", ModelUnknown},
		{"65 chars", strings.Repeat("a", 65), ModelUnknown},
		{"a https URL", "https://proxy.example.com/v1", ModelUnknown},
		{"a bare host", "proxy.example.com/v1", ModelUnknown},
		{"a leading slash", "/v1/chat", ModelUnknown},
		{"a leading dot", ".hidden", ModelUnknown},
		{"a leading hyphen", "-model", ModelUnknown},
		{"spaces", "gpt 6 sol", ModelUnknown},
		{"a newline", "gpt-6-sol\nhost: evil", ModelUnknown},
		{"an at sign", "model@host", ModelUnknown},
		{"a query string", "gpt-6-sol?base=x", ModelUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, got := c.ResolveModel(tc.id)
			if got != tc.want {
				t.Fatalf("ResolveModel(%q) = %v, want %v", tc.id, got, tc.want)
			}
			if got == ModelCustom && m.ID != strings.TrimSpace(tc.id) {
				t.Fatalf("custom model id = %q, want %q", m.ID, strings.TrimSpace(tc.id))
			}
		})
	}
}

func TestInterviewCapable(t *testing.T) {
	c := NewCatalog()

	// A known model WITHOUT interview_brain is refused: we know it and know it is the
	// wrong tool, so saying so beats failing mid-interview.
	var withBrain, withoutBrain string
	for _, m := range c.Models() {
		if m.HasCapability(CapInterviewBrain) && withBrain == "" {
			withBrain = m.ID
		}
		if !m.HasCapability(CapInterviewBrain) && withoutBrain == "" {
			withoutBrain = m.ID
		}
	}
	if withBrain == "" || withoutBrain == "" {
		t.Fatalf("need one model of each kind to test (brain=%q, no-brain=%q)", withBrain, withoutBrain)
	}

	for _, tc := range []struct {
		name    string
		id      string
		wantOK  bool
		wantRes ModelResolution
	}{
		{"catalog model with interview_brain", withBrain, true, ModelKnown},
		{"catalog model without it", withoutBrain, false, ModelKnown},
		// A custom id is allowed: we cannot know what it is, the learner may have a
		// fine-tune built for exactly this, and it is their key. It never gets an AI
		// proposal (t6 §11) and never gets a cost estimate.
		{"a custom id", "ft:gpt-6-sol:org:interviewer", true, ModelCustom},
		{"a malformed id", "https://evil.example/v1", false, ModelUnknown},
		{"empty", "", false, ModelUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ok, res := c.InterviewCapable(tc.id)
			if ok != tc.wantOK || res != tc.wantRes {
				t.Fatalf("InterviewCapable(%q) = (%v, %v), want (%v, %v)", tc.id, ok, res, tc.wantOK, tc.wantRes)
			}
		})
	}
}

func TestEstimateCostMicros(t *testing.T) {
	c := NewCatalog()
	// Sonnet 5 at $2 in / $10 out per MTok (read 2026-10-01).
	m, ok := c.Lookup("claude-sonnet-5")
	if !ok {
		t.Fatal("claude-sonnet-5 missing")
	}
	if m.Price.InputMicrosPerMTok != 2_000_000 || m.Price.OutputMicrosPerMTok != 10_000_000 {
		t.Fatalf("claude-sonnet-5 price = %+v; update this test with the new published price", m.Price)
	}

	// 1 MTok in + 1 MTok out = $2 + $10 = 12_000_000 micros.
	if got := c.EstimateCostMicros("claude-sonnet-5", 1_000_000, 1_000_000); got == nil || *got != 12_000_000 {
		t.Fatalf("1M+1M = %v, want 12000000", deref(got))
	}
	// A realistic turn: 3_000 in, 700 out → 6000 + 7000 = 13_000 micros (1.3 cents).
	if got := c.EstimateCostMicros("claude-sonnet-5", 3_000, 700); got == nil || *got != 13_000 {
		t.Fatalf("3000+700 = %v, want 13000", deref(got))
	}
	// Zero tokens is a real answer (zero cost), NOT unknown.
	if got := c.EstimateCostMicros("claude-sonnet-5", 0, 0); got == nil || *got != 0 {
		t.Fatalf("0+0 = %v, want 0", deref(got))
	}

	// No price → nil, which is what makes the month summary say "custom models not
	// estimated" instead of under-reporting the total as if the turn were free.
	for _, id := range []string{"ft:gpt-6-luna:personal:coach", "totally-unknown", ""} {
		if got := c.EstimateCostMicros(id, 1000, 1000); got != nil {
			t.Fatalf("EstimateCostMicros(%q) = %v, want nil", id, *got)
		}
	}
	// Negative counts are nonsense, never a negative cost that would offset the month.
	if got := c.EstimateCostMicros("claude-sonnet-5", -1, 10); got != nil {
		t.Fatalf("negative input tokens = %v, want nil", *got)
	}
}

func TestDivRoundHalfUp(t *testing.T) {
	for _, tc := range []struct{ n, d, want int64 }{
		{0, 1_000_000, 0},
		{499_999, 1_000_000, 0},
		{500_000, 1_000_000, 1},
		{1_499_999, 1_000_000, 1},
		{1_500_000, 1_000_000, 2},
	} {
		if got := divRoundHalfUp(tc.n, tc.d); got != tc.want {
			t.Errorf("divRoundHalfUp(%d, %d) = %d, want %d", tc.n, tc.d, got, tc.want)
		}
	}
}

// TestOpenAIDefaultMatchesTheSmokeCallOutcome documents the recorded decision rather than
// asserting a preference. m1-10's plan promotes gpt-6-sol to the OpenAI default ONLY if a
// smoke call on the owner's own key proves this release can reach it — a brand-new id can
// be unavailable to an individual account long after it is announced, and a default nobody
// can call would greet every new OpenAI learner with a 404 → F11.
//
// The smoke key (OPENAI_KEY_SMOKE) is an OPTIONAL before-launch item. Without it the
// documented fallback applies: keep gpt-5.6-sol. This test pins whichever of the two the
// catalog claims, so a future change has to be deliberate.
func TestOpenAIDefaultMatchesTheSmokeCallOutcome(t *testing.T) {
	const fallback = "gpt-5.6-sol"
	const promoted = "gpt-6-sol"

	switch OpenAIDefaultModel {
	case fallback:
		if os.Getenv("OPENAI_KEY_SMOKE") != "" {
			t.Log("OPENAI_KEY_SMOKE is set in this environment, but the catalog ships the " +
				"fallback default: the smoke call was not run, or it failed. See the decisions log.")
		}
	case promoted:
		// Promoting it means a smoke call succeeded; the decisions log must say when.
	default:
		t.Fatalf("OpenAIDefaultModel = %q, want %q or %q", OpenAIDefaultModel, fallback, promoted)
	}

	c := NewCatalog()
	// Either way BOTH ids stay selectable — the fallback is about which one is the
	// default, never about hiding the other.
	for _, id := range []string{fallback, promoted} {
		if _, ok := c.Lookup(id); !ok {
			t.Fatalf("%q must stay in the catalog whichever is the default", id)
		}
	}
	if got := c.DefaultModel(store.ProviderOpenAI); got != OpenAIDefaultModel {
		t.Fatalf("catalog OpenAI default = %q, want %q", got, OpenAIDefaultModel)
	}
}

func deref(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}
