// Coach data hooks for the BFF (docs/architecture/api.md). The coach service stores the
// user's provider key ENVELOPE-ENCRYPTED and returns only a masked view; this module
// wraps the masked read, the store/delete/toggle writes, the per-page thread history,
// and the streaming chat (SSE) — never the raw key.
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { API_BASE, ApiRequestError, apiFetch } from "./api";

/** One provider key as the coach service masks it (never the raw secret). `is_default`
 *  marks the provider the coach answers with. */
export interface CoachKey {
  provider: string;
  masked_key: string;
  default_model: string;
  name: string;
  enabled: boolean;
  is_default: boolean;
  tested?: boolean;
}

/** GET /coach/key payload. An account may connect one key per provider (0..2); exactly one
 *  is the default. `connected` is true when at least one key is stored. */
export interface CoachKeyResponse {
  keys: CoachKey[];
  connected: boolean;
  default_provider: string;
}

/** The two coach providers, in display order (each with its key-format hint). */
export type ProviderId = "anthropic" | "openai";
export const COACH_PROVIDERS: { id: ProviderId; label: string; keyHint: string }[] = [
  { id: "anthropic", label: "Anthropic", keyHint: "sk-ant-…" },
  { id: "openai", label: "OpenAI", keyHint: "sk-…" },
];

/** One selectable coach model. */
export interface CoachModelOption {
  id: string;
  label: string;
  hint: string;
  tag?: string;
}

/** Curated coach models per provider (F006) — the ones we recommend for coaching. The
 *  Settings form also offers a Custom… escape hatch for any exact id the provider accepts. */
export const COACH_MODELS: Record<ProviderId, CoachModelOption[]> = {
  anthropic: [
    { id: "claude-opus-5", label: "Opus 5", hint: "Most capable — deep reasoning", tag: "Recommended" },
    { id: "claude-opus-4-8", label: "Opus 4.8", hint: "Capable, lower cost" },
    { id: "claude-sonnet-5", label: "Sonnet 5", hint: "Balanced speed & smarts", tag: "Balanced" },
  ],
  openai: [
    { id: "gpt-5.6-sol", label: "GPT-5.6 Sol", hint: "Deepest reasoning", tag: "Deepest" },
    { id: "gpt-5.6-terra", label: "GPT-5.6 Terra", hint: "Balanced", tag: "Balanced" },
    { id: "gpt-5.6-luna", label: "GPT-5.6 Luna", hint: "Fastest, cheapest", tag: "Fastest" },
  ],
};

/** coachModelLabel renders a model id as its friendly label, falling back to the id. */
export function coachModelLabel(id: string): string {
  for (const list of Object.values(COACH_MODELS)) {
    const m = list.find((x) => x.id === id);
    if (m) return m.label;
  }
  return id;
}

/** useCoachKey fetches the masked coach-key config. Retry is off so the empty state
 *  renders promptly if coach is unavailable. `enabled` lets a caller (the coach panel)
 *  defer the fetch until it is actually opened. */
export function useCoachKey(enabled = true) {
  return useQuery<CoachKeyResponse, ApiRequestError>({
    queryKey: ["coach-key"],
    queryFn: () => apiFetch<CoachKeyResponse>("/coach/key"),
    retry: false,
    staleTime: 30_000,
    enabled,
  });
}

/** The PUT /coach/key body — every mode is keyed to a `provider`:
 *   {provider, key[, default_model, name]} → store/replace that provider's key
 *   {provider, default:true}               → make that provider the default
 *   {provider, enabled}                    → toggle that provider's enabled flag
 *   {provider, default_model|name}         → change that provider's model/name (no key) */
export interface PutCoachKeyBody {
  provider: string;
  key?: string;
  default_model?: string;
  name?: string;
  enabled?: boolean;
  default?: boolean;
}

/** usePutCoachKey stores/replaces a key or toggles enabled, then refreshes the masked
 *  read so the Settings section and coach panel reflect the change. */
export function usePutCoachKey() {
  const qc = useQueryClient();
  return useMutation<CoachKeyResponse, ApiRequestError, PutCoachKeyBody>({
    mutationFn: (body) =>
      apiFetch<CoachKeyResponse>("/coach/key", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["coach-key"] }),
  });
}

/** useDeleteCoachKey removes ONE provider's key (a survivor is promoted to default if the
 *  removed key was the default). */
export function useDeleteCoachKey() {
  const qc = useQueryClient();
  return useMutation<void, ApiRequestError, string>({
    mutationFn: (provider) => apiFetch<void>(`/coach/key?provider=${encodeURIComponent(provider)}`, { method: "DELETE" }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["coach-key"] }),
  });
}

/** One persisted chat turn (role user|assistant). */
export interface CoachMessage {
  role: "user" | "assistant";
  content: string;
  createdAt?: string;
}

/** GET /coach/thread?context= payload. */
export interface CoachThreadResponse {
  context?: string;
  messages: CoachMessage[];
}

/** useCoachThread loads the message history for a page context (enabled by the panel). */
export function useCoachThread(context: string, enabled: boolean) {
  return useQuery<CoachThreadResponse, ApiRequestError>({
    queryKey: ["coach-thread", context],
    queryFn: () => apiFetch<CoachThreadResponse>(`/coach/thread?context=${encodeURIComponent(context)}`),
    enabled: enabled && context !== "",
    retry: false,
    staleTime: 5_000,
  });
}

/** The descriptive page context + the message sent to POST /coach/chat. The behaviour
 *  gate (Socratic vs reviewer) is decided server-side from practice — these fields only
 *  flavour the prompt and identify the thread. */
export interface CoachChatBody {
  context: string;
  kind?: string;
  label?: string;
  problemId?: string;
  problemTitle?: string;
  pattern?: string;
  stage?: string;
  weakArea?: string;
  recentOutcome?: string;
  message: string;
}

/** A coach chat failure the panel routes on: `code` is "no_key" / "key_disabled" /
 *  "provider_auth" (→ Settings), "provider_limited" (the key is fine but the provider
 *  account is out of credit or limited — the key stays enabled), or a generic
 *  transport/provider error. `message` is the server's learner-facing copy when it sent one. */
export class CoachChatError extends Error {
  readonly code: string;
  constructor(code: string, message: string) {
    super(message);
    this.name = "CoachChatError";
    this.code = code;
  }
  /** True when the failure means the key must be (re)added in Settings. */
  get routesToSettings(): boolean {
    return this.code === "no_key" || this.code === "key_disabled" || this.code === "provider_auth";
  }
  /** True when the provider account is out of credit / over a limit (top up and retry). */
  get isProviderLimited(): boolean {
    return this.code === "provider_limited";
  }
}

/**
 * streamCoachChat POSTs a chat turn and streams the coach's reply over SSE, invoking
 * onDelta for each token chunk. It resolves when the stream completes (a `done` frame or
 * EOF) and throws a CoachChatError on a non-2xx envelope (e.g. 409 no_key) or an inline
 * SSE `error` frame (e.g. provider_auth). It is NOT built on apiFetch — that wrapper is
 * JSON-only; SSE needs the raw response body reader.
 */
export async function streamCoachChat(body: CoachChatBody, onDelta: (delta: string) => void, signal?: AbortSignal): Promise<void> {
  const res = await fetch(`${API_BASE}/coach/chat`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json", Accept: "text/event-stream" },
    body: JSON.stringify(body),
    signal,
  });

  if (!res.ok || !res.body) {
    let code = "coach_error";
    let message = `coach chat failed (${res.status})`;
    try {
      const j = (await res.json()) as { error?: { code?: string; message?: string } };
      if (j.error?.code) code = j.error.code;
      if (j.error?.message) message = j.error.message;
    } catch {
      // keep the defaults
    }
    throw new CoachChatError(code, message);
  }

  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buf = "";
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    let nl: number;
    while ((nl = buf.indexOf("\n")) >= 0) {
      const line = buf.slice(0, nl);
      buf = buf.slice(nl + 1);
      if (!line.startsWith("data:")) continue;
      const data = line.slice(5).trim();
      if (!data) continue;
      let frame: { delta?: string; done?: boolean; error?: string; message?: string };
      try {
        frame = JSON.parse(data);
      } catch {
        continue;
      }
      if (frame.error) throw new CoachChatError(frame.error, frame.message ?? "coach error");
      if (typeof frame.delta === "string") onDelta(frame.delta);
      if (frame.done) return;
    }
  }
}

/** Provider display labels for the canonical ids the coach service stores. */
export const PROVIDER_LABELS: Record<string, string> = { openai: "OpenAI", anthropic: "Anthropic" };

/** providerLabel renders a provider id for display, falling back to a capitalised id. */
export function providerLabel(id: string): string {
  return PROVIDER_LABELS[id] ?? (id ? id[0]!.toUpperCase() + id.slice(1) : id);
}
