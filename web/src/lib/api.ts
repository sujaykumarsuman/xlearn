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
 * apiFetch calls the BFF at `${API_BASE}${path}` with cookie credentials and
 * JSON handling. On a non-2xx it decodes the error envelope and throws
 * ApiRequestError. `path` must start with "/".
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
      headers: {
        Accept: "application/json",
        ...(init?.headers ?? {}),
      },
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
