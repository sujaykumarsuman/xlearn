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
	"github.com/sujaykumarsuman/xlearn/internal/platform/auth"
	"github.com/sujaykumarsuman/xlearn/internal/platform/secrets"
)

// claimsCtxKey carries verified JWT claims from requireJWT to the handler.
type claimsCtxKey struct{}

// headerCoachMode is the SERVER-AUTHORITATIVE behaviour gate the gateway sets from
// practice's state (attempt|review|general). It is a gateway→coach header, so a browser
// client can never set it — the spoiler-control gate is not client-trusted (ADR-0007).
const headerCoachMode = "X-Coach-Mode"

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

// keysResponse is the GET /keys payload: the account's masked provider keys (0..2), a
// connected flag, and which provider is the default (empty when none).
type keysResponse struct {
	Keys            []keyViewJSON `json:"keys"`
	Connected       bool          `json:"connected"`
	DefaultProvider string        `json:"default_provider"`
}

func keyView(k store.KeyConfig) keyViewJSON {
	return keyViewJSON{Provider: k.Provider, MaskedKey: k.Masked, DefaultModel: k.DefaultModel, Name: k.Name, Enabled: k.Enabled, IsDefault: k.IsDefault}
}

// keysResponseFrom builds the masked list payload from the stored configs.
func keysResponseFrom(keys []store.KeyConfig) keysResponse {
	views := make([]keyViewJSON, 0, len(keys))
	def := ""
	for _, k := range keys {
		views = append(views, keyView(k))
		if k.IsDefault {
			def = k.Provider
		}
	}
	return keysResponse{Keys: views, Connected: len(views) > 0, DefaultProvider: def}
}

// --- key config handlers ---

// writeKeys reads the account's provider keys and writes the masked list + default.
func (s *Service) writeKeys(w http.ResponseWriter, r *http.Request, accountID string) {
	keys, err := s.store.ListKeys(r.Context(), accountID)
	if err != nil {
		s.mapErr(w, "list keys", err)
		return
	}
	writeJSON(w, http.StatusOK, keysResponseFrom(keys))
}

// handleGetKey: GET /keys — the masked provider keys + connected + default (never a raw key).
func (s *Service) handleGetKey(w http.ResponseWriter, r *http.Request) {
	s.writeKeys(w, r, claimsFrom(r.Context()).Subject)
}

// handlePutKey: PUT /keys — one body, four modes, every mode keyed to a provider:
//
//	{provider, key[, default_model, name]} → store/replace that provider's key (sealed)
//	{provider, default:true}               → make that provider the account's default
//	{provider, enabled}                    → toggle that provider's enabled flag
//	{provider, default_model|name}         → change that provider's model/name (no key)
//
// The gateway never sees the raw key beyond forwarding it; coach seals it here.
func (s *Service) handlePutKey(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	var body struct {
		Provider     string `json:"provider"`
		Key          string `json:"key"`
		DefaultModel string `json:"default_model"`
		Name         string `json:"name"`
		Enabled      *bool  `json:"enabled"`
		Default      bool   `json:"default"`
	}
	if err := decodeJSONStrict(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	provider := strings.ToLower(strings.TrimSpace(body.Provider))
	if !store.ValidProvider(provider) {
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

	switch {
	case rawKey != "":
		// Store / replace this provider's key (validate + seal + mask).
		if len(rawKey) > maxRawKeyLen {
			writeError(w, http.StatusUnprocessableEntity, "invalid_key", "key is too long")
			return
		}
		if model == "" {
			if p, ok := s.providerFor(provider); ok {
				model = p.DefaultModel()
			}
		}
		if name == "" {
			name = model
		}
		encKey, encDataKey, err := s.cipher.Seal([]byte(rawKey))
		if err != nil {
			// Never log the key; the seal error carries no secret bytes.
			s.log.Error("coach: seal key failed", "err", err)
			writeError(w, http.StatusInternalServerError, "internal", "could not store key")
			return
		}
		if _, err := s.store.PutKey(r.Context(), store.KeyConfig{
			AccountID: accountID, Provider: provider, EncKey: encKey, EncDataKey: encDataKey,
			Masked: secrets.Mask(rawKey), DefaultModel: model, Name: name,
		}); err != nil {
			s.mapErr(w, "put key", err)
			return
		}

	case body.Default:
		// Make this provider the account's default.
		if _, err := s.store.SetDefault(r.Context(), accountID, provider); err != nil {
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

// handleDeleteKey: DELETE /keys?provider= — remove one provider's key.
func (s *Service) handleDeleteKey(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	provider := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("provider")))
	if !store.ValidProvider(provider) {
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

// handleThread: GET /threads?context= — the message history for a page context.
func (s *Service) handleThread(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	pageContext := strings.TrimSpace(r.URL.Query().Get("context"))
	if pageContext == "" || len(pageContext) > maxContextLen {
		writeError(w, http.StatusBadRequest, "bad_request", "context is required")
		return
	}
	msgs, err := s.store.ThreadHistory(r.Context(), accountID, pageContext)
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

// --- chat (SSE) ---

// handleChat: POST /chat — stream a coach reply over SSE. It persists the user message +
// the assistant reply to the (account, context) thread, builds the server-side prompt
// from the AUTHORITATIVE mode (X-Coach-Mode header), decrypts the key in memory only for
// the provider call, and zeroes it after. A provider auth failure flips enabled=false so
// the client routes back to Settings. Errors before the first byte are a clean 4xx/5xx;
// after streaming has begun they surface as an SSE `error` event.
func (s *Service) handleChat(w http.ResponseWriter, r *http.Request) {
	accountID := claimsFrom(r.Context()).Subject
	var body struct {
		Context       string `json:"context"`
		Kind          string `json:"kind"`
		Label         string `json:"label"`
		ProblemID     string `json:"problemId"`
		ProblemTitle  string `json:"problemTitle"`
		Pattern       string `json:"pattern"`
		Stage         string `json:"stage"`
		WeakArea      string `json:"weakArea"`
		RecentOutcome string `json:"recentOutcome"`
		Message       string `json:"message"`
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
		message = message[:maxMessageLen]
	}

	// Load the DEFAULT provider's key config (the one the coach answers with). No key /
	// disabled key → a clean 4xx the client maps to the Settings empty state (this is BEFORE
	// any SSE bytes, so a real status is possible).
	kc, err := s.store.GetDefaultKey(r.Context(), accountID)
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
	}
	mode := normalizeMode(r.Header.Get(headerCoachMode), ctx)

	// Persist the user's turn, then build the provider request from the thread history
	// (which now ends with this message).
	threadID, err := s.store.EnsureThread(r.Context(), accountID, pageKey)
	if err != nil {
		s.mapErr(w, "chat: ensure thread", err)
		return
	}
	if err := s.store.AppendMessage(r.Context(), threadID, store.RoleUser, message); err != nil {
		s.mapErr(w, "chat: append user message", err)
		return
	}
	history, err := s.store.ThreadHistory(r.Context(), accountID, pageKey)
	if err != nil {
		s.mapErr(w, "chat: history", err)
		return
	}
	model := kc.DefaultModel
	if model == "" {
		model = provider.DefaultModel()
	}
	req := ChatRequest{Model: model, System: systemPrompt(mode, ctx), Messages: buildTurns(history)}

	// Decrypt the key IN MEMORY ONLY; zero it the moment the provider call returns.
	rawKey, err := s.cipher.Open(kc.EncKey, kc.EncDataKey)
	if err != nil {
		s.log.Error("coach: decrypt key failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "could not use stored key")
		return
	}
	defer secrets.Zero(rawKey)

	callCtx, cancel := context.WithTimeout(r.Context(), providerCallLimit)
	defer cancel()

	sse := newSSEWriter(w)
	var reply strings.Builder
	streamErr := provider.Stream(callCtx, string(rawKey), req, func(delta string) error {
		reply.WriteString(delta)
		return sse.event("", map[string]any{"delta": delta})
	})
	secrets.Zero(rawKey) // zero as soon as the provider call is done (before persistence)

	s.finishChat(r, accountID, kc.Provider, threadID, &reply, sse, streamErr)
}

// finishChat handles the terminal outcome of a chat stream: it persists the assistant
// reply (best-effort, on a detached context so a client disconnect doesn't abort the
// write), and emits the closing SSE event — or, when the provider rejected the key before
// any byte was sent, a clean 4xx + an enabled=false flip.
func (s *Service) finishChat(r *http.Request, accountID, provider, threadID string, reply *strings.Builder, sse *sseWriter, streamErr error) {
	// A detached context so persistence survives a cancelled request (client gone).
	bg, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
	defer cancel()

	switch {
	case streamErr == nil:
		if reply.Len() > 0 {
			if err := s.store.AppendMessage(bg, threadID, store.RoleAssistant, reply.String()); err != nil {
				s.log.Warn("coach: persist assistant reply failed", "err", err)
			}
		}
		_ = sse.event("done", map[string]any{"done": true})

	case errors.Is(streamErr, ErrProviderAuth):
		// The user's key is bad: disable that provider so the client routes back to Settings.
		if err := s.store.SetKeyEnabled(bg, accountID, provider, false); err != nil {
			s.log.Warn("coach: disable bad key failed", "err", err)
		}
		if sse.started {
			_ = sse.event("error", map[string]any{"error": "provider_auth", "message": "your provider key was rejected — re-add it in Settings"})
		} else {
			writeError(sse.w, http.StatusConflict, "provider_auth", "your provider key was rejected — re-add it in Settings")
		}

	default:
		// Other provider/transport error (or client disconnect). Persist any partial reply
		// so the thread isn't left with a dangling user message.
		s.log.Warn("coach: chat stream error", "err", streamErr)
		if reply.Len() > 0 {
			if err := s.store.AppendMessage(bg, threadID, store.RoleAssistant, reply.String()); err != nil {
				s.log.Warn("coach: persist partial reply failed", "err", err)
			}
		}
		if sse.started {
			_ = sse.event("error", map[string]any{"error": "provider_error", "message": "the coach could not complete the reply"})
		} else {
			writeError(sse.w, http.StatusBadGateway, "provider_error", "the coach provider is unavailable")
		}
	}
}

// clampField trims a descriptive field and bounds its length (defence against a hostile
// client stuffing the prompt). The label allows a slightly longer cap than other fields.
func clampField(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > maxLabelLen {
		return s[:maxLabelLen]
	}
	return s
}

// buildTurns maps the stored thread history to provider turns (last historyLimit). It
// aligns the window to begin on a USER turn: a fixed-size tail of an alternating,
// odd-length history can otherwise start on an assistant turn, and Anthropic's
// /v1/messages requires the first message to be role=user (a leading assistant → 400,
// which would fail every turn on an over-length thread permanently). Dropping leading
// non-user turns keeps the request valid without losing the recent user↔assistant pairs.
func buildTurns(history []store.Message) []ChatMessage {
	if len(history) > historyLimit {
		history = history[len(history)-historyLimit:]
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

// requireJWT verifies the gateway-minted JWT (Authorization: Bearer) via JWKS and stores
// its claims in the request context (ADR-0006).
func (s *Service) requireJWT(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearer(r)
		if token == "" {
			writeUnauthenticated(w)
			return
		}
		claims, err := s.verifier.Verify(r.Context(), token)
		if err != nil {
			s.log.Warn("jwt verify failed", "err", err)
			writeUnauthenticated(w)
			return
		}
		ctx := context.WithValue(r.Context(), claimsCtxKey{}, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func claimsFrom(ctx context.Context) auth.Claims {
	c, _ := ctx.Value(claimsCtxKey{}).(auth.Claims)
	return c
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

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

// writeUnauthenticated emits exactly the api.md 401 envelope the SPA redirects on.
func writeUnauthenticated(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"unauthenticated"}}`))
}
