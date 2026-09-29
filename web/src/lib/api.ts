// Typed client seam for the xLearn BFF API. Nothing calls it this sprint — it
// exists so later screens share one fetch wrapper with the error-envelope
// contract from docs/architecture/api.md (same-origin cookie auth, no CORS).

// API_BASE is derived from Vite's base URL so it always matches the mount
// prefix: base "/xlearn/" -> "/xlearn/api/v1". v1 is the canonical 1.0 surface; the
// gateway also serves the unversioned /xlearn/api as a compat alias (ADR-0021).
export const API_BASE = `${import.meta.env.BASE_URL.replace(/\/+$/, "")}/api/v1`;

// REQUEST_TIMEOUT_MS bounds a single BFF call so a hung socket surfaces as an error
// (the screen's error state) instead of an indefinite loading skeleton.
const REQUEST_TIMEOUT_MS = 15_000;

/**
 * coursePathApi is the BFF path of a course-scoped resource: coursePathApi("x",
 * "/dashboard") is "/paths/x/dashboard" (m1-03: course-scoped aggregates live under the
 * existing /paths/{slug}/… prefix, t0 §7). Items stay addressed by global id
 * (/problems/{id}, /mistakes/{id}, …). An unknown or closed course answers
 * 404 course_not_found.
 */
export function coursePathApi(slug: string, rest: string): string {
  return `/paths/${encodeURIComponent(slug)}${rest}`;
}

/** The error object inside the BFF's error envelope. `reason` refines some codes (the
 *  coach's provider_limited); `retry_after` (seconds) rides on a 429; `limit` (bytes) on
 *  a 413 body_too_large (m1-05, ADR-0035 §4). */
export interface ApiError {
  code: string;
  message?: string;
  reason?: string;
  retry_after?: number;
  limit?: number;
  details?: unknown;
}

/** The BFF error envelope: `{ "error": { code, message?, details? } }`. */
export interface ApiErrorEnvelope {
  error: ApiError;
}

/** Thrown for any non-2xx BFF response, carrying the decoded envelope. The server's
 *  `code` and `reason` are kept exactly as sent — a 429 is `rate_limited` (the gateway's
 *  limits), `busy` (the public-profile compose cap), `provider_limited` (the coach) or
 *  identity's `too_many_requests`, and callers route on it — so a limit never collapses
 *  into one generic code. `retryAfter` (seconds) is added from Retry-After or the
 *  envelope's retry_after. */
export class ApiRequestError extends Error {
  readonly status: number;
  readonly code: string;
  readonly reason?: string;
  readonly retryAfter?: number;
  readonly details?: unknown;

  constructor(status: number, error: ApiError, retryAfter?: number) {
    super(error.message ?? error.code);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = error.code;
    this.reason = error.reason;
    this.retryAfter = retryAfter ?? positiveSeconds(error.retry_after);
    this.details = error.details;
  }

  /** True for the unauthenticated case the SPA redirects to OAuth on (S02). */
  get isUnauthenticated(): boolean {
    return this.status === 401 || this.code === "unauthenticated";
  }

  /** True for a 429 (any code): the request was refused for now, not for good. */
  get isRateLimited(): boolean {
    return this.status === 429;
  }

  /** True for a 413: the body was over the server's cap. */
  get isTooLarge(): boolean {
    return this.status === 413;
  }
}

/** positiveSeconds is n when it is a finite number > 0, else undefined. */
function positiveSeconds(n: unknown): number | undefined {
  return typeof n === "number" && Number.isFinite(n) && n > 0 ? Math.ceil(n) : undefined;
}

/** parseRetryAfter reads a Retry-After header (delta-seconds or an HTTP date) as seconds. */
export function parseRetryAfter(value: string | null, now: number = Date.now()): number | undefined {
  if (!value) return undefined;
  const trimmed = value.trim();
  if (/^\d+$/.test(trimmed)) return positiveSeconds(Number(trimmed)) ?? 1;
  const at = Date.parse(trimmed);
  if (Number.isNaN(at)) return undefined;
  return Math.max(1, Math.ceil((at - now) / 1000));
}

/**
 * limitErrorMessage is the retry copy for the existing error states (m1-05): a 429 reads
 * "Too many requests — try again in N s" and a 413 "That's too large to send.". null for
 * any other error, so the caller keeps its own copy.
 */
export function limitErrorMessage(err: unknown): string | null {
  if (!(err instanceof ApiRequestError)) return null;
  if (err.isRateLimited) {
    return err.retryAfter ? `Too many requests — try again in ${err.retryAfter} s.` : "Too many requests — try again in a moment.";
  }
  if (err.isTooLarge) return "That’s too large to send.";
  return null;
}

/** The longest Retry-After a query waits out on its own; a longer one surfaces the error
 *  state (with its retry copy) instead of a spinner. */
const AUTO_RETRY_MAX_WAIT_S = 2;

/**
 * shouldRetryQuery is the shared react-query retry policy: a 429 is retried once, and only
 * when its Retry-After is short (the compose cap's `busy`, Retry-After 1); a 413 is never
 * retried (the same body stays too large); anything else keeps v1's single retry.
 */
export function shouldRetryQuery(failureCount: number, err: unknown): boolean {
  if (err instanceof ApiRequestError) {
    if (err.isRateLimited) return failureCount < 1 && (err.retryAfter ?? Infinity) <= AUTO_RETRY_MAX_WAIT_S;
    if (err.isTooLarge) return false;
  }
  return failureCount < 1;
}

/** queryRetryDelay never retries a 429 sooner than its Retry-After (react-query's default
 *  backoff otherwise: 1 s, 2 s, 4 s … capped at 30 s). */
export function queryRetryDelay(attempt: number, err: unknown): number {
  if (err instanceof ApiRequestError && err.isRateLimited && err.retryAfter) return err.retryAfter * 1000;
  return Math.min(1000 * 2 ** attempt, 30_000);
}

/**
 * requestHeaders builds apiFetch's headers: `Accept: application/json` always, and
 * `Content-Type: application/json` on every non-GET/HEAD call, with or without a body. The
 * gateway refuses a mutating /api call that isn't JSON (415 unsupported_media_type) — the
 * cross-site write guard (m1-04; ADR-0033 §9). A caller's own headers win; a Content-Type
 * the caller already set (any casing) is not doubled.
 */
function requestHeaders(init?: RequestInit): Headers {
  const headers = new Headers(init?.headers);
  if (!headers.has("Accept")) headers.set("Accept", "application/json");
  const method = (init?.method ?? "GET").toUpperCase();
  if (method !== "GET" && method !== "HEAD" && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return headers;
}

/**
 * apiFetch calls the BFF at `${API_BASE}${path}` with cookie credentials and
 * JSON handling (every write is sent as application/json — see requestHeaders).
 * On a non-2xx it decodes the error envelope and throws ApiRequestError. `path`
 * must start with "/".
 */
export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);
  let res: Response;
  try {
    res = await fetch(`${API_BASE}${path}`, {
      credentials: "include",
      ...init,
      signal: init?.signal ?? controller.signal,
      headers: requestHeaders(init),
    });
  } catch (err) {
    // A timeout aborts the request — surface it as an error the screen can render,
    // rather than leaving the caller (and its loading skeleton) hanging forever.
    if (controller.signal.aborted) {
      throw new ApiRequestError(0, { code: "timeout", message: "The request timed out." });
    }
    throw err;
  } finally {
    clearTimeout(timer);
  }

  if (!res.ok) {
    let error: ApiError = { code: "internal", message: res.statusText };
    try {
      const body = (await res.json()) as Partial<ApiErrorEnvelope>;
      if (body.error?.code) {
        error = body.error;
      }
    } catch {
      // Non-JSON error body — keep the status-derived default.
    }
    throw new ApiRequestError(res.status, error, parseRetryAfter(res.headers.get("Retry-After")));
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}
