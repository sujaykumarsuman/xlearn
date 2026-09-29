// 429 / 413 handling in the API client (m1-05, ADR-0035 §4). Every 429 keeps the server's
// own code (and reason): the gateway's `rate_limited` and `busy`, the coach's
// `provider_limited` (settings.ts routes on it), identity's `too_many_requests`. The client
// only ADDS retryAfter, from Retry-After or the envelope's retry_after.
import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiRequestError, apiFetch, limitErrorMessage, parseRetryAfter, queryRetryDelay, shouldRetryQuery } from "./api";
import { restoreFetch } from "../test/fetchMock";

function answer(status: number, body: unknown, headers: Record<string, string> = {}) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async () => new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json", ...headers } })),
  );
}

async function caught(p: Promise<unknown>): Promise<ApiRequestError> {
  try {
    await p;
  } catch (e) {
    if (e instanceof ApiRequestError) return e;
    throw e;
  }
  throw new Error("expected an ApiRequestError");
}

describe("apiFetch 429 / 413", () => {
  afterEach(restoreFetch);

  it("keeps a 429 provider_limited's code and reason, and adds retryAfter from Retry-After", async () => {
    answer(429, { error: { code: "provider_limited", reason: "rate_limit", message: "The provider is rate-limiting this key." } }, { "Retry-After": "20" });
    const err = await caught(apiFetch("/coach/key"));
    expect(err.status).toBe(429);
    expect(err.code).toBe("provider_limited");
    expect(err.reason).toBe("rate_limit");
    expect(err.retryAfter).toBe(20);
    expect(err.isRateLimited).toBe(true);
  });

  it("keeps a 429 busy's code and reads retry_after from the envelope", async () => {
    answer(429, { error: { code: "busy", message: "The server is busy.", retry_after: 1 } });
    const err = await caught(apiFetch("/u/ada"));
    expect(err.code).toBe("busy");
    expect(err.retryAfter).toBe(1);
  });

  it("prefers the Retry-After header and keeps rate_limited as sent", async () => {
    answer(429, { error: { code: "rate_limited", retry_after: 5 } }, { "Retry-After": "6" });
    const err = await caught(apiFetch("/auth/login", { method: "POST" }));
    expect(err.code).toBe("rate_limited");
    expect(err.retryAfter).toBe(6);
  });

  it("keeps a 413 body_too_large", async () => {
    answer(413, { error: { code: "body_too_large", message: "request body too large", limit: 1048576 } });
    const err = await caught(apiFetch("/me", { method: "PATCH", body: "{}" }));
    expect(err.status).toBe(413);
    expect(err.code).toBe("body_too_large");
    expect(err.isTooLarge).toBe(true);
    expect(err.retryAfter).toBeUndefined();
  });
});

describe("limit copy and retry policy", () => {
  it("renders the retry copy for 429 and 413 only", () => {
    expect(limitErrorMessage(new ApiRequestError(429, { code: "rate_limited" }, 12))).toBe("Too many requests — try again in 12 s.");
    expect(limitErrorMessage(new ApiRequestError(429, { code: "busy" }))).toBe("Too many requests — try again in a moment.");
    expect(limitErrorMessage(new ApiRequestError(413, { code: "body_too_large" }))).toBe("That’s too large to send.");
    expect(limitErrorMessage(new ApiRequestError(500, { code: "internal" }))).toBeNull();
    expect(limitErrorMessage(new Error("x"))).toBeNull();
  });

  it("parses Retry-After seconds and HTTP dates", () => {
    expect(parseRetryAfter("7")).toBe(7);
    expect(parseRetryAfter("0")).toBe(1);
    expect(parseRetryAfter(null)).toBeUndefined();
    expect(parseRetryAfter("soon")).toBeUndefined();
    const now = Date.parse("2026-10-12T09:00:00Z");
    expect(parseRetryAfter("Mon, 12 Oct 2026 09:00:30 GMT", now)).toBe(30);
  });

  it("never retries a 429 sooner than Retry-After, and a long one not at all", () => {
    const busy = new ApiRequestError(429, { code: "busy" }, 1);
    expect(shouldRetryQuery(0, busy)).toBe(true);
    expect(shouldRetryQuery(1, busy)).toBe(false);
    expect(queryRetryDelay(0, busy)).toBe(1000);

    const limited = new ApiRequestError(429, { code: "rate_limited" }, 60);
    expect(shouldRetryQuery(0, limited)).toBe(false);
    expect(shouldRetryQuery(0, new ApiRequestError(429, { code: "rate_limited" }))).toBe(false);

    expect(shouldRetryQuery(0, new ApiRequestError(413, { code: "body_too_large" }))).toBe(false);
    // Everything else keeps v1's single retry.
    expect(shouldRetryQuery(0, new ApiRequestError(502, { code: "upstream" }))).toBe(true);
    expect(shouldRetryQuery(1, new ApiRequestError(502, { code: "upstream" }))).toBe(false);
    expect(queryRetryDelay(0, new ApiRequestError(502, { code: "upstream" }))).toBe(1000);
  });
});
