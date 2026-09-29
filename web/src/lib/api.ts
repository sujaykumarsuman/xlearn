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

/** The error object inside the BFF's error envelope. */
export interface ApiError {
  code: string;
  message?: string;
  details?: unknown;
}

/** The BFF error envelope: `{ "error": { code, message?, details? } }`. */
export interface ApiErrorEnvelope {
  error: ApiError;
}

/** Thrown for any non-2xx BFF response, carrying the decoded envelope. */
export class ApiRequestError extends Error {
  readonly status: number;
  readonly code: string;
  readonly details?: unknown;

  constructor(status: number, error: ApiError) {
    super(error.message ?? error.code);
    this.name = "ApiRequestError";
    this.status = status;
    this.code = error.code;
    this.details = error.details;
  }

  /** True for the unauthenticated case the SPA redirects to OAuth on (S02). */
  get isUnauthenticated(): boolean {
    return this.status === 401 || this.code === "unauthenticated";
  }
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
    throw new ApiRequestError(res.status, error);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}
