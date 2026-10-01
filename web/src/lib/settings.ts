// Coach data hooks for the BFF (docs/architecture/api.md). The coach service stores the
// user's provider key ENVELOPE-ENCRYPTED and returns only a masked view; this module
// wraps the masked read, the store/delete/toggle writes, the per-page thread history,
// and the streaming chat (SSE) — never the raw key.
import { useEffect, type RefObject } from "react";
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

/** The features that each have their OWN default key + model (m1-10 / ADR-0031 §7):
 *  the chat coach, and the text interviewer's brain (m6a-02). */
export type CoachFeature = "coach" | "interview";

/** One per-feature default: whose key answers for that feature, and with which model. */
export interface CoachFeatureDefault {
  provider: string;
  model: string;
}

/** Month-to-date spend on the learner's own keys (UTC calendar month), for F13's "This
 *  month on your keys" line. Always an estimate: `has_unknown_cost` says at least one turn
 *  had no published price (a custom model id, or a stream that reported no usage). */
export interface CoachUsageMonth {
  messages: number;
  input_tokens: number;
  output_tokens: number;
  est_cost_micros: number;
  has_unknown_cost: boolean;
}

/** GET /coach/key payload. An account may connect one key per provider (0..2); exactly one
 *  is the default. `connected` is true when at least one key is stored.
 *
 *  `defaults` and `usage_month` arrived with m1-10 and are optional here on purpose: coach
 *  and the gateway deploy independently, so a freshly-rolled SPA can briefly talk to a
 *  v1.6.0 coach that doesn't send them. Every reader treats absent as "unset" rather than
 *  assuming the fields exist. */
export interface CoachKeyResponse {
  keys: CoachKey[];
  connected: boolean;
  default_provider: string;
  defaults?: Partial<Record<CoachFeature, CoachFeatureDefault | null>>;
  usage_month?: CoachUsageMonth;
}

/** The two coach providers, in display order (each with its key-format hint). */
export type ProviderId = "anthropic" | "openai";
export const COACH_PROVIDERS: { id: ProviderId; label: string; keyHint: string }[] = [
  { id: "anthropic", label: "Anthropic", keyHint: "sk-ant-…" },
  { id: "openai", label: "OpenAI", keyHint: "sk-…" },
];

/** What a catalog model may be used FOR. `chat` is every entry; `interview_brain` may back
 *  the interview default (m6a-02); `voice_shell` is the realtime voice shell (M6b) and is
 *  provisional — no chat model carries it today. */
export type CoachCapability = "chat" | "interview_brain" | "voice_shell";

/** A model's published list price, in MICROS per million tokens ($2.00/MTok = 2_000_000).
 *  Integer micros, so a month of turns can't accumulate float drift. */
export interface CoachModelPrice {
  input_micros_per_mtok: number;
  output_micros_per_mtok: number;
}

/** One entry of the SERVER model catalog. */
export interface CoachModel {
  id: string;
  provider: string;
  label: string;
  capabilities: CoachCapability[];
  /** The published list price. Never null on the wire; null only on a row this SPA
   *  synthesises for an id the catalog doesn't carry (uncatalogedModel), which is what the
   *  UI shows as "cost unknown". */
  price: CoachModelPrice | null;
  as_of: string;
  recommended: boolean;
  /** A model whose provider retains the conversation for 30 days. Allowed on a BYO key —
   *  it is the learner's own provider account — so it is LABELLED, never refused. */
  covered_model: boolean;
}

/** GET /coach/models payload: the dated catalog, the providers this build can actually
 *  call, and each provider's default model id. */
export interface CoachModelsResponse {
  as_of: string;
  providers: string[];
  models: CoachModel[];
  defaults: Record<string, string>;
}

/** The shape of a model id this release doesn't know but will still accept — a fine-tune,
 *  a dated snapshot, a provider alias. It deliberately rejects anything URL-shaped, which
 *  is what keeps "pick a model" from becoming "pick an endpoint" (there is no base-URL
 *  field anywhere). The server enforces the same regex; this is the inline-error copy's
 *  trigger, not the guard. */
export const COACH_MODEL_ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$/;

/**
 * useCoachModels fetches the server model catalog. v1 baked the list into the SPA, so a
 * provider shipping or retiring a model needed a frontend release; it now comes from coach
 * (m1-10 task 2). Cached for an hour because the catalog only changes with a coach
 * release, and retry is off so a failure falls through promptly to the degraded view
 * (F12: "on catalog failure the menu shows only the currently set model").
 */
export function useCoachModels(enabled = true) {
  return useQuery<CoachModelsResponse, ApiRequestError>({
    queryKey: ["coach-models"],
    queryFn: () => apiFetch<CoachModelsResponse>("/coach/models"),
    retry: false,
    staleTime: 60 * 60 * 1000,
    enabled,
  });
}

/** coachModelLabel renders a model id as its catalog label, falling back to the id itself —
 *  which is exactly what a custom id (and any id a stale catalog has dropped) should show. */
export function coachModelLabel(id: string, models?: CoachModel[]): string {
  return models?.find((m) => m.id === id)?.label ?? id;
}

/** The capability tags F12 renders beside a model, in the board's order. A capability with
 *  no tag (`chat`, which every entry has) is not shown. */
export const COACH_CAPABILITY_TAGS: { capability: CoachCapability; tag: string }[] = [
  { capability: "interview_brain", tag: "Interview" },
  { capability: "voice_shell", tag: "Voice" },
];

/** coachCapabilityTags lists a model's visible tags ("Interview", "Voice"). */
export function coachCapabilityTags(model: CoachModel): string[] {
  return COACH_CAPABILITY_TAGS.filter((t) => model.capabilities.includes(t.capability)).map((t) => t.tag);
}

/** coachPriceLine renders a catalog price as AB01 F12 writes it: "$2 in · $10 out". Whole
 *  dollars print bare, anything else to cents ("$0.10 in · $0.50 out"). */
export function coachPriceLine(price: CoachModelPrice): string {
  return `${dollars(price.input_micros_per_mtok)} in · ${dollars(price.output_micros_per_mtok)} out`;
}

/** coachPriceCaption is F12's dated price caption: "Prices per MTok, as of Sep 24, 2026".
 *  The date is the catalog's own `as_of` — the UI always shows it beside the numbers,
 *  because this estimates the learner's spend on their own key, it is not a bill. */
export function coachPriceCaption(asOf: string): string {
  return `Prices per MTok, as of ${monthDayYear(asOf)}`;
}

/** coachUsageLine is F13's month-to-date figures: "212 messages · 1.4 M tokens · ≈ $3.10
 *  (estimate)". The "· custom models not estimated" suffix is NOT part of this string —
 *  the board appends it outside the bold span, so the caller renders it. */
export function coachUsageLine(usage: CoachUsageMonth): string {
  const tokens = (usage.input_tokens + usage.output_tokens) / 1_000_000;
  const messages = `${usage.messages} ${usage.messages === 1 ? "message" : "messages"}`;
  return `${messages} · ${tokens.toFixed(1)} M tokens · ≈ ${dollars(usage.est_cost_micros)} (estimate)`;
}

/** dollars renders micros as a currency amount, to cents unless it is a whole dollar. */
function dollars(micros: number): string {
  const value = micros / 1_000_000;
  return `$${Number.isInteger(value) ? value : value.toFixed(2)}`;
}

/** monthDayYear renders an ISO date (the catalog's `as_of`) as "Sep 24, 2026". It is parsed
 *  and formatted in UTC: the catalog date is a calendar date, not an instant, so a learner
 *  west of Greenwich must not see it slide to the day before. */
function monthDayYear(iso: string): string {
  const d = new Date(`${iso}T00:00:00Z`);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" });
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

/**
 * coachFeatureDefault resolves which key + model answers for one feature.
 *
 * `coach` falls back to the v1 shape (the `is_default` key, else the first one) so the
 * panel still works against a coach that predates per-feature defaults, and so an account
 * whose key_default row hasn't been back-filled isn't told its coach is off. `interview`
 * has NO fallback: unset is its normal starting state, and guessing a model for it could
 * silently put a non-interview model behind an interview.
 */
export function coachFeatureDefault(data: CoachKeyResponse | undefined, feature: CoachFeature): CoachFeatureDefault | null {
  const explicit = data?.defaults?.[feature];
  if (explicit) return explicit;
  if (feature !== "coach") return null;
  const key = data?.keys?.find((k) => k.is_default) ?? data?.keys?.[0];
  return key ? { provider: key.provider, model: key.default_model } : null;
}

/** The PUT /coach/key body — every mode is keyed to a `provider`:
 *   {provider, key[, default_model, name]}            → store/replace that provider's key
 *   {provider, default:true[, feature, default_model]} → point a feature's default at it
 *   {provider, enabled}                                → toggle that provider's enabled flag
 *   {provider, default_model|name}                     → change that provider's model/name (no key)
 *
 *  `feature` ("coach" | "interview") only applies to the default mode; omitted means
 *  "coach", which is what every v1.6.0 client sent. The server decodes strictly, so an
 *  unknown field (a `base_url`, say) is a 400 — never a silently honoured endpoint. */
export interface PutCoachKeyBody {
  provider: string;
  key?: string;
  default_model?: string;
  name?: string;
  enabled?: boolean;
  default?: boolean;
  feature?: CoachFeature;
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

/** Why the provider refused, as the server classifies it (m1-10 task 4). The panel maps
 *  each one to its own AB01 line, which is the whole point of the taxonomy: a quota
 *  failure, a model the key may not use and an unsupported region all arrived as one
 *  "provider_limited" in v1, so the learner was told to top up a funded account. */
export type CoachErrorReason = "auth" | "quota" | "rate_limit" | "model_access" | "region" | "unavailable";

/** A coach chat failure the panel routes on: `code` is "no_key" / "key_disabled" /
 *  "provider_auth" (→ Settings), "provider_limited" (the key is fine but the provider
 *  account is out of credit or limited — the key stays enabled), or a generic
 *  transport/provider error. `message` is the server's learner-facing copy when it sent one.
 *  `reason` is the finer m1-10 classification; it is absent from a v1.6.0 coach's reply, so
 *  it falls back to "unavailable" and the panel's generic line. */
export class CoachChatError extends Error {
  readonly code: string;
  readonly reason: CoachErrorReason;
  constructor(code: string, message: string, reason?: string) {
    super(message);
    this.name = "CoachChatError";
    this.code = code;
    this.reason = isCoachErrorReason(reason) ? reason : "unavailable";
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

const COACH_ERROR_REASONS: readonly string[] = ["auth", "quota", "rate_limit", "model_access", "region", "unavailable"];

/** isCoachErrorReason narrows a server-sent reason, so an unknown one (a newer coach
 *  classifying something this SPA has never heard of) degrades to the generic line rather
 *  than rendering a blank note. */
function isCoachErrorReason(v: unknown): v is CoachErrorReason {
  return typeof v === "string" && COACH_ERROR_REASONS.includes(v);
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
    let reason: string | undefined;
    try {
      const j = (await res.json()) as { error?: { code?: string; message?: string; reason?: string } };
      if (j.error?.code) code = j.error.code;
      if (j.error?.message) message = j.error.message;
      reason = j.error?.reason;
    } catch {
      // keep the defaults
    }
    throw new CoachChatError(code, message, reason);
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
      let frame: { delta?: string; done?: boolean; error?: string; message?: string; reason?: string };
      try {
        frame = JSON.parse(data);
      } catch {
        continue;
      }
      if (frame.error) throw new CoachChatError(frame.error, frame.message ?? "coach error", frame.reason);
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

/**
 * catalogModelsFor lists one provider's catalog models, with the currently-set model
 * appended when the catalog doesn't carry it.
 *
 * That tail case covers both a custom id (never in the catalog) and a catalog we could not
 * read at all — F12: "on catalog failure the menu shows only the currently set model". The
 * learner can always see what they are on, and we never invent a label or a price we don't
 * have. It also keeps a model a later catalog has dropped visible while it is still in use.
 */
export function catalogModelsFor(models: CoachModel[] | undefined, provider: string, selected: string): CoachModel[] {
  const list = (models ?? []).filter((m) => m.provider === provider);
  if (selected === "" || list.some((m) => m.id === selected)) return list;
  return [...list, uncatalogedModel(provider, selected)];
}

/** uncatalogedModel is a row for a model id the catalog doesn't carry: its id as its label,
 *  no capability tags, and no price — a price we don't have is rendered as nothing, never
 *  as "$0", because "cost unknown" is the honest statement. */
export function uncatalogedModel(provider: string, id: string): CoachModel {
  return { id, provider, label: id, capabilities: ["chat"], price: null, as_of: "", recommended: false, covered_model: false };
}

/** usePopoverDismiss closes an open popover on Escape or a pointerdown outside `ref`. The
 *  ref must wrap the TRIGGER as well as the panel, or clicking the trigger to close would
 *  dismiss on pointerdown and immediately re-open on click. */
export function usePopoverDismiss(ref: RefObject<HTMLElement | null>, open: boolean, close: () => void) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && close();
    const onPointer = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) close();
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);
}
