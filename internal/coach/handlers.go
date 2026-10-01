package coach

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sujaykumarsuman/xlearn/internal/coach/store"
	"github.com/sujaykumarsuman/xlearn/internal/course"
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// headerCoachMode is the SERVER-AUTHORITATIVE behaviour gate the gateway sets from
// practice's state (attempt|review|general). It is a gateway→coach header, so a browser
// client can never set it — the spoiler-control gate is not client-trusted (ADR-0007).
const headerCoachMode = "X-Coach-Mode"

// The other gateway→coach chat headers (m1-07), equally server-authoritative:
//
//   - X-Coach-Course is the course the gateway resolved for the page (a problem's
//     path_slug, a course page's course, "" for an account-wide page). It picks the prompt
//     persona only (coursePersona); the thread's label still comes from the context and
//     ?path= (threadCourse).
//   - X-Coach-Attempt is the open attempt's id, sent only when a D27 assist is recorded on
//     it. It is stored on both rows of the turn (attempt_id); a value that isn't a UUID is
//     ignored, never refused.
const (
	headerCoachCourse  = "X-Coach-Course"
	headerCoachAttempt = "X-Coach-Attempt"
)

// Input caps guard the free-text fields so a hostile client can't stash unbounded text
// or spend the user's key on a giant prompt.
const (
	maxRawKeyLen      = 512
	maxContextLen     = 200
	maxLabelLen       = 300
	maxMessageLen     = 4000
	maxDescFieldLen   = 300
	providerCallLimit = 120 * time.Second
)

// --- JSON response shapes ---

// keyViewJSON is the masked, safe-to-return view of ONE provider key config (NEVER the raw
// key or sealed material). is_default marks the provider the coach answers with.
type keyViewJSON struct {
	Provider     string `json:"provider"`
	MaskedKey    string `json:"masked_key"`
	DefaultModel string `json:"default_model"`
	Name         string `json:"name"`
	Enabled      bool   `json:"enabled"`
	IsDefault    bool   `json:"is_default"`
}

// featureDefaultJSON is one per-feature default: which provider's key answers for that
// feature and which model it uses. Absent (null) when the feature has no default — for
// `interview` that is the normal starting state, rendered "Not set" (AB01 F13).
type featureDefaultJSON struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// usageMonthJSON is the account's month-to-date spend on its own keys, for F13's "This
// month on your keys" line. Display only, and always an estimate: HasUnknownCost says at
// least one turn had no published price (a custom model id, or a stream that reported no
// usage), which the UI surfaces as "· custom models not estimated" so the number is never
// mistaken for a total.
type usageMonthJSON struct {
	Messages       int64 `json:"messages"`
	InputTokens    int64 `json:"input_tokens"`
	OutputTokens   int64 `json:"output_tokens"`
	EstCostMicros  int64 `json:"est_cost_micros"`
	HasUnknownCost bool  `json:"has_unknown_cost"`
}

// keysResponse is the GET /keys payload. The v1 shape (keys[].is_default,
// default_provider) is PRESERVED so a v1.6.0 tab left open across the upgrade keeps
// working — but both are now derived from key_default(feature='coach'), never from the
// dead api_key_config.is_default column. m1-10 adds the per-feature defaults and the
// month-to-date usage.
type keysResponse struct {
	Keys            []keyViewJSON `json:"keys"`
	Connected       bool          `json:"connected"`
	DefaultProvider string        `json:"default_provider"`

	// Defaults is keyed by feature ("coach", "interview"); a feature with no default maps
	// to null rather than being omitted, so the client can tell "not set" from "this
	// release doesn't know about that feature".
	Defaults   map[string]*featureDefaultJSON `json:"defaults"`
	UsageMonth usageMonthJSON                 `json:"usage_month"`
}

func keyView(k store.KeyConfig) keyViewJSON {
	return keyViewJSON{Provider: k.Provider, MaskedKey: k.Masked, DefaultModel: k.DefaultModel, Name: k.Name, Enabled: k.Enabled, IsDefault: k.IsDefault}
}

// keysResponseFrom builds the masked list payload from the stored configs, the account's
// per-feature defaults and its month-to-date usage.
func keysResponseFrom(keys []store.KeyConfig, defaults []store.FeatureDefault, usage store.Usage) keysResponse {
	views := make([]keyViewJSON, 0, len(keys))
	def := ""
	for _, k := range keys {
		views = append(views, keyView(k))
		if k.IsDefault {
			def = k.Provider
		}
	}
	// Every known feature is present as a key, null when unset.
	byFeature := map[string]*featureDefaultJSON{
		store.FeatureCoach:     nil,
		store.FeatureInterview: nil,
	}
	for _, d := range defaults {
		byFeature[d.Feature] = &featureDefaultJSON{Provider: d.Provider, Model: d.Model}
	}
	return keysResponse{
		Keys:            views,
		Connected:       len(views) > 0,
		DefaultProvider: def,
		Defaults:        byFeature,
		UsageMonth: usageMonthJSON{
			Messages:       usage.Messages,
			InputTokens:    usage.InputTokens,
			OutputTokens:   usage.OutputTokens,
			EstCostMicros:  usage.EstCostMicros,
			HasUnknownCost: usage.HasUnknownCost,
		},
	}
}

// --- key config handlers ---

// writeKeys reads the account's provider keys, per-feature defaults and month-to-date
// usage, and writes the masked payload.
func (s *Service) writeKeys(w http.ResponseWriter, r *http.Request, accountID string) {
	keys, err := s.store.ListKeys(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "list keys", err)
		return
	}
	defaults, err := s.store.ListDefaults(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "list defaults", err)
		return
	}
	usage, err := s.store.UsageMonth(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "usage month", err)
		return
	}
	writeJSON(w, http.StatusOK, keysResponseFrom(keys, defaults, usage))
}

// --- model catalog ---

// modelJSON is one catalog entry as served.
type modelJSON struct {
	ID           string   `json:"id"`
	Provider     string   `json:"provider"`
	Label        string   `json:"label"`
	Capabilities []string `json:"capabilities"`
	Price        Price    `json:"price"`
	AsOf         string   `json:"as_of"`
	Recommended  bool     `json:"recommended"`
	CoveredModel bool     `json:"covered_model"`
}

// handleModels: GET /models — the dated server catalog (ADR-0031 §7, t5 §9).
//
// This replaces the model list v1 baked into the SPA, which could only be corrected by a
// frontend release. It carries no learner data at all, but stays behind the same JWT gate
// as every other coach route so the public profile remains the ONLY unauthenticated /api
// route.
func (s *Service) handleModels(w http.ResponseWriter, r *http.Request) {
	c := s.catalog
	// ChatModels, not Models: this endpoint feeds the coach/interview model pickers, so a
	// non-chat entry (a realtime voice shell, when m6b-01 adds one) must never be offered
	// here as something a learner can answer chats with.
	chat := c.ChatModels()
	models := make([]modelJSON, 0, len(chat))
	for _, m := range chat {
		// Only offer models whose provider this build actually has a client for, so the
		// catalog can never advertise something a chat would then fail on.
		if _, ok := s.providerFor(m.Provider); !ok {
			continue
		}
		caps := m.Capabilities
		if caps == nil {
			caps = []string{}
		}
		models = append(models, modelJSON{
			ID: m.ID, Provider: m.Provider, Label: m.Label, Capabilities: caps,
			Price: m.Price, AsOf: m.AsOf, Recommended: m.Recommended, CoveredModel: m.CoveredModel,
		})
	}
	defaults := map[string]string{}
	for _, id := range s.ProviderIDs() {
		if d := c.DefaultModel(id); d != "" {
			defaults[id] = d
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"as_of":     c.AsOf(),
		"providers": s.ProviderIDs(),
		"models":    models,
		"defaults":  defaults,
	})
}

// handleGetKey: GET /keys — the masked provider keys + connected + default (never a raw key).
func (s *Service) handleGetKey(w http.ResponseWriter, r *http.Request) {
	s.writeKeys(w, r, auth.ClaimsFrom(r.Context()).Subject)
}

// handlePutKey: PUT /keys — one body, four modes, every mode keyed to a provider:
//
//	{provider, key[, default_model, name]}      → store/replace that provider's key (sealed)
//	{provider, default:true[, feature, model]}  → make that provider a feature's default
//	{provider, enabled}                         → toggle that provider's enabled flag
//	{provider, default_model|name}              → change that provider's model/name (no key)
//
// m1-10 adds `feature` to the default mode: "coach" (the implicit default) or "interview"
// (the text interviewer's brain, m6a-02). Omitted means "coach", which is what every
// v1.6.0 client sends.
//
// The gateway never sees the raw key beyond forwarding it; coach seals it here, into both
// sealed pairs at once (sealKey).
func (s *Service) handlePutKey(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	var body struct {
		Provider     string `json:"provider"`
		Key          string `json:"key"`
		DefaultModel string `json:"default_model"`
		Name         string `json:"name"`
		Enabled      *bool  `json:"enabled"`
		Default      bool   `json:"default"`
		Feature      string `json:"feature"`
	}
	// Strict decoding (DisallowUnknownFields) is load-bearing, not tidiness: it is what
	// makes a `base_url` field a 400 rather than a silently ignored SSRF attempt. Coach
	// calls provider endpoints this build was configured with, never one from a request.
	if err := decodeJSONStrict(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(body.Provider))
	// Validate against the Service's provider REGISTRY, not a static list: a provider this
	// build has no client for must not be storable, or the key would be unusable.
	if _, ok := s.providerFor(provider); !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid_provider", "provider must be openai or anthropic")
		return
	}
	rawKey := strings.TrimSpace(body.Key)
	model := strings.TrimSpace(body.DefaultModel)
	name := strings.TrimSpace(body.Name)
	if len(model) > maxDescFieldLen {
		writeError(w, http.StatusUnprocessableEntity, "invalid_model", "default model is too long")
		return
	}
	if len(name) > maxDescFieldLen {
		writeError(w, http.StatusUnprocessableEntity, "invalid_name", "name is too long")
		return
	}
	feature := strings.ToLower(strings.TrimSpace(body.Feature))
	if feature == "" {
		feature = store.FeatureCoach
	}
	if !store.ValidFeature(feature) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_feature", "feature must be coach or interview")
		return
	}

	switch {
	case rawKey != "":
		// Store / replace this provider's key (validate + seal + mask).
		if len(rawKey) > maxRawKeyLen {
			writeError(w, http.StatusUnprocessableEntity, "invalid_key", "key is too long")
			return
		}
		if model == "" {
			model = s.catalog.DefaultModel(provider)
		}
		if model == "" {
			if p, ok := s.providerFor(provider); ok {
				model = p.DefaultModel()
			}
		}
		if !s.validModelForKey(w, model) {
			return
		}
		if name == "" {
			name = model
		}
		sealed, err := s.sealKey(accountID, provider, rawKey)
		if err != nil {
			// Never log the key; the seal error carries no secret bytes.
			s.log.Error("coach: seal key failed", "err", err)
			writeError(w, http.StatusInternalServerError, "internal", "could not store key")
			return
		}
		if _, err := s.store.PutKey(r.Context(), store.KeyConfig{
			AccountID: accountID, Provider: provider,
			EncKey: sealed.EncKey, EncDataKey: sealed.EncDataKey,
			EncKeyAD: sealed.EncKeyAD, EncDataKeyAD: sealed.EncDataKeyAD,
			KEKID: sealed.KEKID, ADSrcDigest: sealed.ADSrcDigest,
			Masked: secrets.Mask(rawKey), DefaultModel: model, Name: name,
		}); err != nil {
			s.mapErr(w, "put key", err)
			return
		}

	case body.Default:
		// Point one feature's default at this provider's key.
		//
		// An omitted model means "use the key's own default_model" — but that model must
		// be validated for the FEATURE too, not waved through. Validating only what the
		// client sent let `{"provider":"openai","default":true,"feature":"interview"}`
		// store a chat-only model as the interview brain (every new OpenAI key's
		// default_model is one), which is precisely what the 422 below exists to prevent.
		// So resolve the effective model here and validate that, rather than leaving the
		// store to fill a blank nobody checked.
		if model == "" {
			cur, err := s.store.GetKey(r.Context(), accountID, provider)
			if err != nil {
				s.mapErr(w, "set default: get key", err)
				return
			}
			model = cur.DefaultModel
		}
		if model != "" && !s.validModelForFeature(w, feature, model) {
			return
		}
		if _, err := s.store.SetDefault(r.Context(), accountID, provider, feature, model); err != nil {
			s.mapErr(w, "set default", err)
			return
		}

	case body.Enabled != nil:
		// Toggle this provider's enabled flag without touching the key.
		if err := s.store.SetKeyEnabled(r.Context(), accountID, provider, *body.Enabled); err != nil {
			s.mapErr(w, "set enabled", err)
			return
		}

	case model != "" || name != "":
		// Meta update: switch model / rename this provider's coach without re-entering the key.
		// A partial edit keeps the untouched field; an empty name falls back to the model id.
		if model != "" && !s.validModelForKey(w, model) {
			return
		}
		cur, err := s.store.GetKey(r.Context(), accountID, provider)
		if err != nil {
			s.mapErr(w, "meta: get key", err)
			return
		}
		if model == "" {
			model = cur.DefaultModel
		}
		if name == "" {
			name = model
		}
		if _, err := s.store.UpdateKeyMeta(r.Context(), accountID, provider, model, name); err != nil {
			s.mapErr(w, "update meta", err)
			return
		}

	default:
		writeError(w, http.StatusBadRequest, "bad_request", "provide a key, default_model/name, enabled, or default")
		return
	}

	s.writeKeys(w, r, accountID)
}

// validModelForKey rejects a malformed model id (422 `unknown_model`) and accepts
// everything else: a catalog entry, or a well-formed id this release has never heard of,
// which is stored as-is and shown "cost unknown". It writes the error response and
// reports false when the request should stop.
//
// Accepting unknown-but-well-formed ids is deliberate. The catalog goes stale between
// releases, learners have fine-tunes (`ft:gpt-6-luna:org:suffix`) and dated snapshots, and
// it is their key and their money — refusing an id we simply don't recognise would make
// every provider release a blocker on ours.
func (s *Service) validModelForKey(w http.ResponseWriter, model string) bool {
	if _, res := s.catalog.ResolveModel(model); res == ModelUnknown {
		writeError(w, http.StatusUnprocessableEntity, "unknown_model", "that isn't a model id")
		return false
	}
	return true
}

// validModelForFeature validates a model against what the FEATURE needs.
//
// For `interview` (t6 §11, T5 §9 key/catalog row):
//   - a catalog model carrying interview_brain → accepted;
//   - a catalog model WITHOUT it → 422 `model_not_interview_capable`. We know this model
//     and know it is the wrong tool, so refusing now beats failing mid-interview;
//   - a well-formed custom id → accepted. We cannot know what it is, the learner may have
//     a fine-tune built for exactly this, and it is their key. It never gets an AI
//     proposal (t6 §11) and never gets a cost estimate;
//   - anything else → 422 `unknown_model`.
//
// For `coach`, any model that passes validModelForKey is fine.
func (s *Service) validModelForFeature(w http.ResponseWriter, feature, model string) bool {
	if feature != store.FeatureInterview {
		return s.validModelForKey(w, model)
	}
	ok, res := s.catalog.InterviewCapable(model)
	switch {
	case res == ModelUnknown:
		writeError(w, http.StatusUnprocessableEntity, "unknown_model", "that isn't a model id")
		return false
	case !ok:
		writeError(w, http.StatusUnprocessableEntity, "model_not_interview_capable", "that model can't run the interviewer")
		return false
	}
	return true
}

// handleDeleteKey: DELETE /keys?provider= — remove one provider's key.
func (s *Service) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	if _, ok := s.providerFor(provider); !ok {
		writeError(w, http.StatusUnprocessableEntity, "invalid_provider", "provider must be openai or anthropic")
		return
	}
	if err := s.store.DeleteKey(r.Context(), accountID, provider); err != nil {
		s.mapErr(w, "delete key", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- thread history ---

// handleThread: GET /threads?context= — the message history for a page context. The
// context is normalized to its thread key first (course.NormalizeCoachContext), so a v1
// form from an open v1.6.0 tab reads the same thread as its `<course>:` form. The
// response echoes the context as sent (the v1 shape).
func (s *Service) handleThread(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	pageContext := strings.TrimSpace(r.URL.Query().Get("context"))
	if pageContext == "" || len(pageContext) > maxContextLen {
		writeError(w, http.StatusBadRequest, "bad_request", "context is required")
		return
	}
	msgs, err := s.store.ThreadHistory(r.Context(), accountID, s.courses.NormalizeCoachContext(pageContext).Key)
	if err != nil {
		s.mapErr(w, "thread history", err)
		return
	}
	out := make([]map[string]any, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, map[string]any{
			"role":      m.Role,
			"content":   m.Content,
			"createdAt": m.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"context": pageContext, "messages": out})
}

// --- page contexts (sprint m1-03; t0 §7) ---

// threadCourse is the course a chat's thread and its messages are labelled with
// (coach_thread.path_slug / coach_message.path_slug; "" writes NULL):
//
//   - a course-scoped context: its course, i.e. the `<course>:` prefix, or DefaultSlug
//     for a v1 form (the parser's PathSlug);
//   - problem:<id>: the item's course, which only curriculum knows, so the gateway passes
//     it as POST /chat?path=<slug>. No value (a v1.6.0 gateway during a rolling update)
//     or a slug the registry doesn't know is DefaultSlug: v1 was DSA-only. A label never
//     fails a chat, so an unknown value is logged, not refused;
//   - account-wide (catalog, settings, general) or unrecognised: "" (NULL).
func (s *Service) threadCourse(cc course.CoachContext, pathParam string) string {
	switch {
	case cc.CourseScoped():
		return cc.PathSlug
	case cc.Kind == course.CoachKindProblem:
		if _, ok := s.courses.Lookup(pathParam); ok {
			return pathParam
		}
		if pathParam != "" {
			s.log.Warn("coach: chat ?path= names no known course; labelling the problem thread with the default course", "path", pathParam)
		}
		return course.DefaultSlug
	default:
		return ""
	}
}

// --- chat (SSE) ---

// handleChat: POST /chat[?path=<slug>] — stream a coach reply over SSE. The body's
// context is normalized to its thread key (course.NormalizeCoachContext: a course-scoped
// context is `<course>:<ctx>`, and a v1 form maps to DefaultSlug's), and the thread is
// labelled with its course (threadCourse; ?path= matters only for a problem context).
// The body is v1's (plus the gateway's live_items in general mode): the course never
// travels in it. It persists the user message + the assistant reply to the (account,
// context) thread, builds the server-side prompt (coach-prompt@2) from the AUTHORITATIVE
// mode (X-Coach-Mode) and the course persona (X-Coach-Course), decrypts the key in memory
// only for the provider call, and zeroes it after. A rejected key flips enabled=false so
// the client routes back to Settings; an out-of-credit or rate-limited provider account
// keeps the key and tells the learner to top up. Errors before the first byte are a clean
// 4xx/5xx; after streaming has begun they surface as an SSE `error` event.
//
// The order of the refusals (m1-07):
//
//	locked mode            409 coach_paused        (defensive; nothing parsed or taken)
//	body                   400 bad_request
//	key lookup             409 no_key / key_disabled, 500 unknown provider
//	L18 (admitChat)        429 coach_busy → coach_rate_limited → coach_daily_cap
//	persist, history, key  500, after giving the token and the day's message back
//
// so a request refused at any step persists nothing, and L18 runs before the user turn is
// persisted and before the key is decrypted.
func (s *Service) handleChat(w http.ResponseWriter, r *http.Request) {
	accountID := auth.ClaimsFrom(r.Context()).Subject
	rawMode := strings.TrimSpace(r.Header.Get(headerCoachMode))
	if strings.EqualFold(rawMode, modeLocked) {
		// The gateway answers locked itself and never forwards it; if one ever arrives,
		// refuse it the same way rather than letting normalizeMode turn it into a mode.
		writePaused(w)
		return
	}

	var body struct {
		Context       string         `json:"context"`
		Kind          string         `json:"kind"`
		Label         string         `json:"label"`
		ProblemID     string         `json:"problemId"`
		ProblemTitle  string         `json:"problemTitle"`
		Pattern       string         `json:"pattern"`
		Stage         string         `json:"stage"`
		WeakArea      string         `json:"weakArea"`
		RecentOutcome string         `json:"recentOutcome"`
		Message       string         `json:"message"`
		LiveItems     []liveItemJSON `json:"live_items"`
	}
	if err := decodeJSONLoose(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	pageKey := strings.TrimSpace(body.Context)
	message := strings.TrimSpace(body.Message)
	if pageKey == "" || len(pageKey) > maxContextLen {
		writeError(w, http.StatusBadRequest, "bad_request", "context is required")
		return
	}
	if message == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "message is required")
		return
	}
	if len(message) > maxMessageLen {
		message = strings.ToValidUTF8(message[:maxMessageLen], "")
	}

	// Load the DEFAULT provider's key config (the one the coach answers with). No key /
	// disabled key → a clean 4xx the client maps to the Settings empty state (this is BEFORE
	// any SSE bytes, so a real status is possible).
	kc, err := s.store.GetDefaultKey(r.Context(), accountID, store.FeatureCoach)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusConflict, "no_key", "no provider key configured")
		return
	}
	if err != nil {
		s.mapErr(w, "chat: get key", err)
		return
	}
	if !kc.Enabled {
		writeError(w, http.StatusConflict, "key_disabled", "the provider key is disabled")
		return
	}
	provider, ok := s.providerFor(kc.Provider)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal", "unknown provider")
		return
	}

	// L18: streams → per minute → per UTC day. A refusal has already been written and has
	// consumed nothing. The stream slot is held until this handler returns — the stream
	// ended, failed, or the client went.
	adm, ok := s.admitChat(w, r, accountID)
	if !ok {
		return
	}
	defer adm.done()

	// Descriptive (non-authoritative) context — flavours the prompt only. The behaviour
	// gate is the mode header, so lying in these fields can't flip a spoiler-free attempt.
	ctx := pageContext{
		Kind:          clampField(body.Kind),
		Label:         clampField(body.Label),
		ProblemID:     clampField(body.ProblemID),
		ProblemTitle:  clampField(body.ProblemTitle),
		Pattern:       clampField(body.Pattern),
		Stage:         clampField(body.Stage),
		WeakArea:      clampField(body.WeakArea),
		RecentOutcome: clampField(body.RecentOutcome),
		LiveItems:     liveItemsFrom(body.LiveItems),
	}
	mode := normalizeMode(rawMode, ctx)
	// Both rows of this turn record how it was produced (migration 00007).
	meta := store.MessageMeta{PromptV: PromptVersion, AttemptID: attemptHeader(r)}

	// Persist the user's turn, then build the provider request from the thread's recent
	// history (which now ends with this message). Any failure from here until the provider
	// is called gives the token and the day's message back: the turn spent nothing.
	cc := s.courses.NormalizeCoachContext(pageKey)
	threadID, err := s.store.EnsureThread(r.Context(), accountID, cc.Key, s.threadCourse(cc, r.URL.Query().Get("path")))
	if err != nil {
		adm.refund(context.WithoutCancel(r.Context()))
		s.mapErr(w, "chat: ensure thread", err)
		return
	}
	if err := s.store.AppendMessage(r.Context(), threadID, store.RoleUser, message, meta); err != nil {
		adm.refund(context.WithoutCancel(r.Context()))
		s.mapErr(w, "chat: append user message", err)
		return
	}
	history, err := s.store.RecentMessages(r.Context(), threadID, historyLimit)
	if err != nil {
		adm.refund(context.WithoutCancel(r.Context()))
		s.mapErr(w, "chat: history", err)
		return
	}
	// The model comes from the COACH feature's default row first (what the switcher
	// writes), then the key's own default_model, then the catalog/provider default.
	model := kc.FeatureModel
	if model == "" {
		model = kc.DefaultModel
	}
	if model == "" {
		model = s.catalog.DefaultModel(kc.Provider)
	}
	if model == "" {
		model = provider.DefaultModel()
	}
	persona := coursePersona(s.courses, strings.TrimSpace(r.Header.Get(headerCoachCourse)))
	req := ChatRequest{Model: model, System: systemPrompt(persona, mode, ctx), Messages: buildTurns(history)}

	// Decrypt the key IN MEMORY ONLY; zero it the moment the provider call returns.
	// openKey prefers the AD-bound pair and falls back to the legacy pair only when the AD
	// pair is absent or stale (see keycrypto.go).
	rawKey, err := s.openKey(kc)
	if err != nil {
		adm.refund(context.WithoutCancel(r.Context()))
		s.log.Error("coach: decrypt key failed", "err", err, "provider", kc.Provider)
		writeError(w, http.StatusInternalServerError, "internal", "could not use stored key")
		return
	}
	defer secrets.Zero(rawKey)

	callCtx, cancel := context.WithTimeout(r.Context(), providerCallLimit)
	defer cancel()

	sse := newSSEWriter(w)
	var reply strings.Builder
	result, streamErr := provider.Stream(callCtx, string(rawKey), req, func(delta string) error {
		reply.WriteString(delta)
		return sse.event("", map[string]any{"delta": delta})
	})
	secrets.Zero(rawKey) // zero as soon as the provider call is done (before persistence)

	s.finishChat(r, accountID, kc.Provider, model, threadID, meta, &reply, sse, result, streamErr)
}

// attemptHeader returns X-Coach-Attempt, lower-cased, when it is a canonical UUID, and ""
// otherwise: the attempt id only labels the turn's rows, so a malformed value is dropped
// (stored NULL), never a reason to refuse the chat.
func attemptHeader(r *http.Request) string {
	v := strings.ToLower(strings.TrimSpace(r.Header.Get(headerCoachAttempt)))
	if len(v) != 36 {
		return ""
	}
	for i, c := range v {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return ""
			}
		default:
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				return ""
			}
		}
	}
	return v
}

// liveItemJSON is one entry of the gateway's live_items (general mode). The id is a
// problem id; it is accepted as a JSON string or number so a type slip in a producer can
// never 400 every general-mode chat.
type liveItemJSON struct {
	ID    flexString `json:"id"`
	Title string     `json:"title"`
}

// flexString decodes a JSON string or number as a string (anything else is "").
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*f = flexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		*f = flexString(n.String())
		return nil
	}
	*f = ""
	return nil
}

// turnUsage builds what to store on the assistant message from the stream's outcome.
// est_cost_micros is NULL unless both the catalog knows the model's price and the stream
// actually reported token counts — a cost we cannot compute is recorded as unknown rather
// than as zero, which is what keeps the month-to-date line honest.
func (s *Service) turnUsage(provider, model string, result StreamResult) store.MessageUsage {
	u := store.MessageUsage{
		Provider:     provider,
		Model:        model,
		StopReason:   result.StopReason,
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		HasTokens:    result.Usage.HasTokens,
	}
	if result.Usage.HasTokens {
		u.EstCostMicros = s.catalog.EstimateCostMicros(model, result.Usage.InputTokens, result.Usage.OutputTokens)
	}
	return u
}

// Learner-facing copy for the provider outcomes the coach explains in the panel.
const (
	msgProviderAuth    = "your provider key was rejected — re-add it in Settings"
	msgProviderLimited = "your provider account is out of credit or limited — top up and retry"
	// truncationNote is appended to a reply the provider cut off, both in the live stream
	// and in the persisted thread, so a reload shows the same thing and the next turn's
	// history tells the model its last answer was incomplete.
	truncationNote = "⚠️ This reply was cut off at the length limit — ask me to continue."
)

// finishChat handles the terminal outcome of a chat stream: it persists the assistant
// reply (best-effort, on a detached context so a client disconnect doesn't abort the
// write), and emits the closing SSE event. Before any byte was sent it can still write a
// clean 4xx: a rejected key (409, and the key is disabled) or an out-of-credit / limited
// provider account (429, and the key stays enabled).
func (s *Service) finishChat(r *http.Request, accountID, provider, model, threadID string, meta store.MessageMeta, reply *strings.Builder, sse *sseWriter, result StreamResult, streamErr error) {
	// A detached context so persistence survives a cancelled request (client gone).
	bg, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
	defer cancel()

	usage := s.turnUsage(provider, model, result)

	// persistReply saves whatever reply text there is — with the turn's usage — so the
	// thread isn't left with a dangling user message after a partial stream, and a
	// partial turn is still counted against the month (the learner was billed for it).
	persistReply := func() {
		if reply.Len() == 0 {
			return
		}
		if err := s.store.AppendAssistantMessage(bg, threadID, reply.String(), usage, meta); err != nil {
			s.log.Warn("coach: persist assistant reply failed", "err", err)
		}
	}

	// errorEvent emits the terminal failure. The v1 `error` code and `message` are kept
	// verbatim so an open v1.6.0 tab keeps rendering them; `reason` is the finer m1-10
	// classification the current SPA maps to AB01's copy (F10 quota/rate_limit, F11
	// model_access, F15b auth, and F10's region variant).
	errorEvent := func(status int, code, message string) {
		reason := ErrorReason(streamErr)
		if sse.started {
			_ = sse.event("error", map[string]any{"error": code, "message": message, "reason": reason})
		} else {
			writeJSON(sse.w, status, map[string]any{
				"error": map[string]any{"code": code, "message": message, "reason": reason},
			})
		}
	}

	switch {
	case streamErr == nil:
		if result.Truncated {
			s.log.Info("coach: reply truncated", "provider", provider, "model", model, "stop_reason", result.StopReason)
			note := truncationNote
			if reply.Len() > 0 {
				note = "\n\n" + note
			}
			reply.WriteString(note)
			_ = sse.event("", map[string]any{"delta": note})
		}
		persistReply()
		_ = sse.event("done", map[string]any{"done": true, "truncated": result.Truncated})

	case errors.Is(streamErr, ErrProviderAuth):
		// The key ITSELF is bad: disable that provider so the client routes back to
		// Settings (AB01 F15b). This is the ONLY kind that disables a key.
		if err := s.store.SetKeyEnabled(bg, accountID, provider, false); err != nil {
			s.log.Warn("coach: disable bad key failed", "err", err)
		}
		errorEvent(http.StatusConflict, "provider_auth", msgProviderAuth)

	case errors.Is(streamErr, ErrProviderLimited):
		// The key is fine; the account is out of credit, rate limited, barred from this
		// model, or in an unsupported region. Keep the key ENABLED — disabling it would
		// make the learner re-enter a working key after topping up, which is exactly the
		// v1 bug this taxonomy fixes. `reason` tells the SPA which line to show.
		s.log.Info("coach: provider account limited",
			"provider", provider, "model", model, "reason", ErrorReason(streamErr), "err", streamErr)
		persistReply()
		errorEvent(http.StatusTooManyRequests, "provider_limited", msgProviderLimited)

	default:
		// Other provider/transport error (or client disconnect).
		s.log.Warn("coach: chat stream error", "err", streamErr)
		persistReply()
		if sse.started {
			errorEvent(0, "provider_error", "the coach could not complete the reply")
		} else {
			errorEvent(http.StatusBadGateway, "provider_error", "the coach provider is unavailable")
		}
	}
}

// clampField makes a descriptive field one trimmed line and bounds its length (defence
// against a hostile client stuffing the prompt). Collapsing whitespace keeps a field on its
// own CONTEXT line, so a value with newlines can't pose as a section of the prompt (a
// fake MODE line); the cut never splits a UTF-8 sequence.
func clampField(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > maxLabelLen {
		return strings.ToValidUTF8(s[:maxLabelLen], "")
	}
	return s
}

// buildTurns maps the stored thread history to provider turns, within L18's history cap:
// the last historyLimit messages, then the oldest dropped until their content totals at
// most historyMaxBytes — ALWAYS keeping the last message, the current user turn.
//
// It aligns the window to begin on a USER turn: a fixed-size tail of an alternating,
// odd-length history can otherwise start on an assistant turn, and Anthropic's
// /v1/messages requires the first message to be role=user (a leading assistant → 400,
// which would fail every turn on an over-length thread permanently). Dropping leading
// non-user turns keeps the request valid without losing the recent user↔assistant pairs.
func buildTurns(history []store.Message) []ChatMessage {
	if len(history) > historyLimit {
		history = history[len(history)-historyLimit:]
	}
	total := 0
	for _, m := range history {
		total += len(m.Content)
	}
	for len(history) > 1 && total > historyMaxBytes {
		total -= len(history[0].Content)
		history = history[1:]
	}
	for len(history) > 0 && history[0].Role != store.RoleUser {
		history = history[1:]
	}
	turns := make([]ChatMessage, 0, len(history))
	for _, m := range history {
		turns = append(turns, ChatMessage{Role: providerRole(m.Role), Content: m.Content})
	}
	return turns
}

// --- SSE writer ---

// sseWriter lazily writes the text/event-stream response: the 200 header + streaming
// headers are written on the first event, so an early provider rejection can still be a
// clean 4xx. Flush + the disabled write deadline go through http.ResponseController,
// which reaches the base ResponseWriter via httpx.statusWriter's Unwrap.
type sseWriter struct {
	w       http.ResponseWriter
	rc      *http.ResponseController
	started bool
}

func newSSEWriter(w http.ResponseWriter) *sseWriter {
	s := &sseWriter{w: w, rc: http.NewResponseController(w)}
	// Clear the server WriteTimeout (httpx.NewServer sets 60s) up front — NOT only in
	// start(). A provider call may run up to providerCallLimit (120s) before producing
	// its outcome; if it errors BEFORE any SSE frame, start() never ran, so without this
	// the 60s write deadline would have fired and finishChat's clean 4xx/5xx (e.g. the
	// 409 provider_auth the SPA routes to Settings on) could not be written. Reaches the
	// base writer via httpx.statusWriter's Unwrap; a nil error is fine in tests where the
	// recorder has no deadline support.
	_ = s.rc.SetWriteDeadline(time.Time{})
	return s
}

func (s *sseWriter) start() {
	if s.started {
		return
	}
	s.started = true
	h := s.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	// Disable proxy response buffering (nginx honours this; harmless for Traefik).
	h.Set("X-Accel-Buffering", "no")
	// No write deadline for a long-lived stream (the server's default WriteTimeout would
	// otherwise cut it). Reaches the base writer via Unwrap.
	_ = s.rc.SetWriteDeadline(time.Time{})
	s.w.WriteHeader(http.StatusOK)
	_ = s.rc.Flush()
}

// event writes one SSE frame (optional event name + a JSON data payload) and flushes.
// Returns the write error (e.g. the client disconnected) so the caller can stop.
func (s *sseWriter) event(name string, payload any) error {
	s.start()
	var b strings.Builder
	if name != "" {
		b.WriteString("event: ")
		b.WriteString(name)
		b.WriteByte('\n')
	}
	data, _ := json.Marshal(payload)
	b.WriteString("data: ")
	b.Write(data)
	b.WriteString("\n\n")
	if _, err := io.WriteString(s.w, b.String()); err != nil {
		return err
	}
	return s.rc.Flush()
}

// --- shared HTTP helpers ---

// mapErr maps a store error to the right HTTP status + error envelope.
func (s *Service) mapErr(w http.ResponseWriter, what string, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "not found")
	default:
		s.log.Error("coach store error", "op", what, "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "internal error")
	}
}

func decodeJSONStrict(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func decodeJSONLoose(r *http.Request, v any) error {
	return json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"code": code, "message": message}})
}
