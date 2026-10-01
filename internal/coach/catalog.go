package coach

import (
	"regexp"
	"strings"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
)

// The SERVER-SIDE model catalog (ADR-0031 §7, t5 §9 "Providers and catalog").
//
// v1 baked a model list into the SPA, which went stale the moment a provider shipped or
// retired a model and could only be fixed by a frontend release. The catalog now lives
// here and is served by GET /models (GET /api/coach/models through the gateway), so the
// switcher, the Settings panel and the onboarding step all render whatever this release
// knows — and a learner can always type a model id this release has never heard of.
//
// Three rules the catalog exists to enforce:
//
//   - NO USER-SUPPLIED BASE URLS. A learner picks a model id, never an endpoint. The
//     request decoders reject unknown fields, so a `base_url` is a 400 rather than an
//     SSRF vector pointed at the cluster.
//   - PRICES ARE DATED, NEVER GUESSED. Every price here was read off the provider's own
//     pricing page on CatalogAsOf and is carried with that date. The UI always shows the
//     date beside the number, because this is an estimate of the learner's own spend on
//     their own key, not a bill.
//   - A MODEL ERROR IS NEVER A DISABLE. An id this release doesn't know is accepted as a
//     custom model and shown "cost unknown"; a model the key may not use comes back as
//     ErrModelAccess ("pick another model"). Only an auth failure disables a key.
//
// Catalog drift is expected and deliberately cheap: ids and prices change under us, which
// is why nothing here is load-bearing for correctness. The worst a stale entry can do is
// mis-estimate a cost line or offer a model the provider 404s — which surfaces as F11's
// "pick another model", not as a broken coach.

// CatalogAsOf is the date every price and capability in this catalog was read off the
// providers' own pricing and model pages. It is served as `as_of` and rendered beside the
// prices ("Prices per MTok, as of …"), so a learner can see how fresh the estimate is.
const CatalogAsOf = "2026-10-01"

// Model capabilities. A capability says what a model may be used FOR, which is what the
// per-feature defaults validate against (t5 §9, t6 §11).
const (
	// CapChat — usable as the chat coach. Every catalog entry has it.
	CapChat = "chat"
	// CapInterviewBrain — may back the `interview` feature default, i.e. drive the TEXT
	// interviewer (m6a-02). t6 §11 dropped the earlier "realtime-capable" rule: a text
	// interview needs a capable reasoning model, not a realtime one.
	CapInterviewBrain = "interview_brain"
	// CapVoiceShell — may act as the realtime VOICE shell (M6b). The capability is
	// defined here so the contract and AB01's tag legend exist before m6b-01 needs them,
	// but NO ENTRY IN THIS CATALOG CARRIES IT, and that is a decision rather than an
	// oversight. See the voice_shell note below the entries.
	CapVoiceShell = "voice_shell"
)

// Price is a model's published list price, in MICROS per million tokens — i.e. $2.00 per
// MTok is 2_000_000. Integer micros keep the cost arithmetic exact (no float drift across
// a month of messages) and match coach_message.est_cost_micros.
//
// It is the BASE list price only. Cache reads, batch, service tiers and regional
// multipliers all reprice a real call, so a stored estimate is a floor on the input side
// and the UI never presents it as a bill.
type Price struct {
	InputMicrosPerMTok  int64 `json:"input_micros_per_mtok"`
	OutputMicrosPerMTok int64 `json:"output_micros_per_mtok"`
}

// Model is one catalog entry.
type Model struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	Label        string   `json:"label"`
	Capabilities []string `json:"capabilities"`
	Price        Price    `json:"price"`
	AsOf         string   `json:"as_of"`
	// Recommended marks the provider's default pick, badged "Recommended" in the UI.
	// Exactly one per provider.
	Recommended bool `json:"recommended"`
	// CoveredModel marks a model whose provider retains the conversation for 30 days.
	// Covered Models are ALLOWED on a BYO key — it is the learner's own provider account,
	// and blocking them would be us overriding their org's choice — so they are LABELLED,
	// not refused (t5 §9 correction). None of today's entries is one: the Covered Models
	// are Anthropic's Fable and Mythos families, which coach does not list.
	CoveredModel bool `json:"covered_model"`
}

// HasCapability reports whether the model carries c.
func (m Model) HasCapability(c string) bool {
	for _, got := range m.Capabilities {
		if got == c {
			return true
		}
	}
	return false
}

// catalogModels is the catalog itself, provider-grouped and in the order the UI lists it.
//
// Prices read 2026-10-01 from platform.claude.com/docs/en/about-claude/pricing (and each
// model's own overview page) and developers.openai.com/api/docs/pricing. Every number is
// transcribed, none derived.
//
// Which entries carry CapInterviewBrain is catalog data, not a property the providers
// publish. The rule applied: the CURRENT-GENERATION flagship reasoning models, excluding
// the cheap/fast tier and the superseded generation — an interview has to hold a long
// adaptive conversation and judge answers, which is exactly what the fast tier trades
// away. AB01 F12's drawn sample agrees on every model it happens to show.
var catalogModels = []Model{
	// --- Anthropic ---
	{
		ID: "claude-sonnet-5", Provider: store.ProviderAnthropic, Label: "Sonnet 5",
		Capabilities: []string{CapChat, CapInterviewBrain},
		Price:        Price{InputMicrosPerMTok: 2_000_000, OutputMicrosPerMTok: 10_000_000},
		AsOf:         CatalogAsOf, Recommended: true,
	},
	{
		ID: "claude-opus-5-5", Provider: store.ProviderAnthropic, Label: "Opus 5.5",
		Capabilities: []string{CapChat, CapInterviewBrain},
		Price:        Price{InputMicrosPerMTok: 4_000_000, OutputMicrosPerMTok: 20_000_000},
		AsOf:         CatalogAsOf,
	},
	{
		ID: "claude-opus-5", Provider: store.ProviderAnthropic, Label: "Opus 5",
		Capabilities: []string{CapChat, CapInterviewBrain},
		Price:        Price{InputMicrosPerMTok: 5_000_000, OutputMicrosPerMTok: 25_000_000},
		AsOf:         CatalogAsOf,
	},
	{
		// Legacy on Anthropic's own pages ("still available", migrate recommended), kept
		// because v1 offered it and a learner may have it set. No interview_brain: the
		// previous generation is not an interview brain.
		ID: "claude-opus-4-8", Provider: store.ProviderAnthropic, Label: "Opus 4.8",
		Capabilities: []string{CapChat},
		Price:        Price{InputMicrosPerMTok: 5_000_000, OutputMicrosPerMTok: 25_000_000},
		AsOf:         CatalogAsOf,
	},

	// --- OpenAI ---
	{
		ID: "gpt-6-sol", Provider: store.ProviderOpenAI, Label: "GPT-6 Sol",
		Capabilities: []string{CapChat, CapInterviewBrain},
		Price:        Price{InputMicrosPerMTok: 2_000_000, OutputMicrosPerMTok: 10_000_000},
		AsOf:         CatalogAsOf,
	},
	{
		ID: "gpt-6-astra", Provider: store.ProviderOpenAI, Label: "GPT-6 Astra",
		Capabilities: []string{CapChat, CapInterviewBrain},
		Price:        Price{InputMicrosPerMTok: 10_000_000, OutputMicrosPerMTok: 50_000_000},
		AsOf:         CatalogAsOf,
	},
	{
		ID: "gpt-6-luna", Provider: store.ProviderOpenAI, Label: "GPT-6 Luna",
		Capabilities: []string{CapChat},
		Price:        Price{InputMicrosPerMTok: 100_000, OutputMicrosPerMTok: 500_000},
		AsOf:         CatalogAsOf,
	},
	{
		// The OpenAI default this release ships with (see OpenAIDefaultModel).
		ID: "gpt-5.6-sol", Provider: store.ProviderOpenAI, Label: "GPT-5.6 Sol",
		Capabilities: []string{CapChat},
		Price:        Price{InputMicrosPerMTok: 4_000_000, OutputMicrosPerMTok: 20_000_000},
		AsOf:         CatalogAsOf, Recommended: true,
	},
	{
		ID: "gpt-5.6-terra", Provider: store.ProviderOpenAI, Label: "GPT-5.6 Terra",
		Capabilities: []string{CapChat},
		Price:        Price{InputMicrosPerMTok: 2_000_000, OutputMicrosPerMTok: 12_000_000},
		AsOf:         CatalogAsOf,
	},
	{
		ID: "gpt-5.6-luna", Provider: store.ProviderOpenAI, Label: "GPT-5.6 Luna",
		Capabilities: []string{CapChat},
		Price:        Price{InputMicrosPerMTok: 200_000, OutputMicrosPerMTok: 1_200_000},
		AsOf:         CatalogAsOf,
	},
}

// --- why no entry carries voice_shell (spk-04 → m1-10) ---
//
// S6 has reported, so the plan's wording "provisional until S6" is out of date: the realtime
// shell is settled as `gpt-live-1` (D42) with `gpt-realtime-2.1-mini` beside it, and
// spk-04's hand-off asks for both to carry CapVoiceShell here. They are deliberately NOT
// added in this sprint, for two reasons that are about correctness, not effort:
//
//  1. THEIR PRICE CANNOT BE STATED IN THIS SHAPE. On 2026-09-26 `gpt-live-1` was
//     $0.05/MINUTE, billed per second. Price above is micros per MILLION TOKENS. Putting a
//     per-minute number in a per-MTok field would quietly make every cost estimate and the
//     "This month on your keys" total wrong; leaving the price empty would show
//     "cost unknown" for a model whose price IS published. The `billing_shape`,
//     `session_cap_s` and `min_tier` fields that express it are m6b-01's catalog extension,
//     by that hand-off's own account.
//
//  2. THIS CATALOG IS THE COACH/INTERVIEW MODEL PICKER. Everything served by GET /models is
//     offered as a coach model (AB01 F12 lists a provider's models), so a realtime shell
//     listed here is a model a learner can select and then have every chat fail on. AB01
//     agrees: "No chat model carries Voice today; GPT-Live-1 is the provisional shell."
//
// ChatModels below is the guard that makes adding them safe later: the coach surfaces are
// filtered to CapChat, so a voice entry cannot become a selectable coach model by accident.
// The hand-off is re-pointed at m6b-01 in docs/v2/status.md with the price fact preserved.

// Per-provider defaults — the model a key gets when the learner names none, and the entry
// badged "Recommended".
//
// AnthropicDefaultModel is claude-sonnet-5: the balanced current flagship, and what the
// provider client has defaulted to since v1.
const AnthropicDefaultModel = "claude-sonnet-5"

// OpenAIDefaultModel is gpt-5.6-sol.
//
// m1-10's plan makes gpt-6-sol the OpenAI default ONLY if a smoke call on the owner's own
// key proves this release can actually reach it (a brand-new model id can be unavailable
// to an individual account long after it is announced, and a default nobody can call
// would greet every new OpenAI learner with a 404 → F11). That smoke call needs the
// owner's key in OPENAI_KEY_SMOKE, an OPTIONAL before-launch item which was NOT provided
// for this run, so the plan's documented fallback applies: keep gpt-5.6-sol.
//
// gpt-6-sol is in the catalog and freely selectable; it is only not the default. Promoting
// it later is a one-line change plus a smoke call — see the decisions log in
// docs/v2/status.md.
const OpenAIDefaultModel = "gpt-5.6-sol"

// customModelID is the accepted shape of a model id this release does not know: a leading
// alphanumeric then up to 63 of alphanumerics, dot, underscore, colon and hyphen. It
// admits real-world ids coach cannot enumerate — fine-tunes (`ft:gpt-6-luna:org:suffix`),
// dated snapshots, provider aliases — while rejecting anything URL-shaped, which is what
// keeps "pick a model" from becoming "pick an endpoint".
var customModelID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)

// Catalog is the resolved model catalog a Service serves and validates against.
type Catalog struct {
	asOf     string
	models   []Model
	byID     map[string]Model
	defaults map[string]string // provider → default model id
}

// NewCatalog builds the catalog this release ships.
func NewCatalog() *Catalog {
	c := &Catalog{
		asOf:   CatalogAsOf,
		models: catalogModels,
		byID:   make(map[string]Model, len(catalogModels)),
		defaults: map[string]string{
			store.ProviderAnthropic: AnthropicDefaultModel,
			store.ProviderOpenAI:    OpenAIDefaultModel,
		},
	}
	for _, m := range catalogModels {
		c.byID[m.ID] = m
	}
	return c
}

// AsOf is the date the catalog's prices were read.
func (c *Catalog) AsOf() string { return c.asOf }

// Models returns every entry, in catalog order.
func (c *Catalog) Models() []Model { return c.models }

// ChatModels returns only the entries a learner may pick as a coach model — those carrying
// CapChat — in catalog order.
//
// Today that is every entry, so this looks redundant. It is the guard for the moment it
// stops being redundant: when m6b-01 adds the realtime voice shells (`gpt-live-1`,
// `gpt-realtime-2.1-mini`), everything GET /models serves is offered in the coach model
// picker, so without this filter a learner could select a voice shell as their coach model
// and every chat would fail with a provider error. Keeping the coach surfaces on this
// method means that lands as a no-op instead of an outage.
func (c *Catalog) ChatModels() []Model {
	out := make([]Model, 0, len(c.models))
	for _, m := range c.models {
		if m.HasCapability(CapChat) {
			out = append(out, m)
		}
	}
	return out
}

// Defaults maps provider → default model id.
func (c *Catalog) Defaults() map[string]string { return c.defaults }

// DefaultModel is a provider's default model id ("" for an unknown provider).
func (c *Catalog) DefaultModel(provider string) string { return c.defaults[provider] }

// Lookup returns a catalog entry by id.
func (c *Catalog) Lookup(id string) (Model, bool) {
	m, ok := c.byID[id]
	return m, ok
}

// ModelResolution is the outcome of validating a model id the client sent.
type ModelResolution int

const (
	// ModelUnknown — the id is not in the catalog and not even well-formed. 422
	// `unknown_model`. This is the arm a pasted URL lands in.
	ModelUnknown ModelResolution = iota
	// ModelKnown — a catalog entry. Its capabilities and price apply.
	ModelKnown
	// ModelCustom — well-formed but absent from the catalog. Accepted, and shown
	// "cost unknown": no price, and no cost estimate stored for its turns.
	//
	// "Custom" is DERIVED at read time from "absent from the catalog", never stored as a
	// flag. That matters because the catalog changes between releases: a fine-tune stays
	// custom forever, while an id that was custom because this release hadn't heard of it
	// becomes known — with a price — the moment a later release lists it, with no
	// migration and no stale boolean to contradict the catalog (m6a-02 relies on this
	// being derivable straight from key_default.model).
	ModelCustom
)

// ResolveModel classifies a model id (t5 §9 / ADR-0031 §7 validation rules).
func (c *Catalog) ResolveModel(id string) (Model, ModelResolution) {
	id = strings.TrimSpace(id)
	if m, ok := c.byID[id]; ok {
		return m, ModelKnown
	}
	if customModelID.MatchString(id) {
		return Model{ID: id, Label: id}, ModelCustom
	}
	return Model{}, ModelUnknown
}

// InterviewCapable reports whether a model id may back the `interview` default:
//   - a catalog entry carrying CapInterviewBrain → yes;
//   - a catalog entry WITHOUT it → no (422 `model_not_interview_capable`): we know this
//     model and know it is the wrong tool, so saying so beats letting the learner find out
//     mid-interview;
//   - a well-formed custom id → yes. We cannot know what it is, the learner may well have
//     a fine-tune built for exactly this, and it is their key. It never gets an AI
//     proposal (t6 §11) and never gets a cost estimate;
//   - a malformed id → no (422 `unknown_model`).
func (c *Catalog) InterviewCapable(id string) (ok bool, resolution ModelResolution) {
	m, res := c.ResolveModel(id)
	switch res {
	case ModelKnown:
		return m.HasCapability(CapInterviewBrain), res
	case ModelCustom:
		return true, res
	default:
		return false, res
	}
}

// EstimateCostMicros returns the estimated cost of one turn in micros, or nil when no
// estimate is possible — an id outside the catalog (no published price) or a stream that
// reported no usage. A nil result is what makes the month summary say "custom models not
// estimated" rather than quietly under-reporting.
//
// It is integer arithmetic throughout: micros-per-MTok × tokens / 1e6, rounded half up,
// so a month of turns cannot accumulate float drift.
func (c *Catalog) EstimateCostMicros(modelID string, inputTokens, outputTokens int) *int64 {
	m, ok := c.byID[modelID]
	if !ok {
		return nil
	}
	if inputTokens < 0 || outputTokens < 0 {
		return nil
	}
	const perMTok = 1_000_000
	cost := divRoundHalfUp(m.Price.InputMicrosPerMTok*int64(inputTokens), perMTok) +
		divRoundHalfUp(m.Price.OutputMicrosPerMTok*int64(outputTokens), perMTok)
	return &cost
}

// divRoundHalfUp divides two non-negative integers, rounding halves up.
func divRoundHalfUp(n, d int64) int64 { return (n + d/2) / d }
