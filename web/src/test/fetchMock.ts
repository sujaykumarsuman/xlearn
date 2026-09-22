import { vi } from "vitest";

/** A minimal route handler: returns the status + JSON body for a request, or a raw
 *  Response (e.g. an SSE stream from sseResponse) that is passed through verbatim. */
export type RouteHandler = (url: string, init?: RequestInit) => { status: number; body?: unknown } | Response;

/** installFetchMock stubs global fetch with a route handler. Returns the vi mock
 *  so tests can assert on calls. Pair with restoreFetch() in afterEach. */
export function installFetchMock(handler: RouteHandler) {
  const fn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const raw = typeof input === "string" ? input : input.toString();
    // The client calls the versioned surface (/api/v1/…); the gateway aliases it onto
    // /api/… (ADR-0021). Normalise here so the route handlers can keep matching the
    // unversioned paths regardless of which surface the client uses.
    const url = raw.replace("/api/v1/", "/api/");
    const result = handler(url, init);
    if (result instanceof Response) return result;
    const { status, body } = result;
    return new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    });
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

/** sseResponse builds a text/event-stream Response whose body streams the given frames
 *  (each a full "data: {...}\n\n" chunk) — the coach chat SSE test double. */
export function sseResponse(frames: string[], status = 200): Response {
  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      const enc = new TextEncoder();
      for (const f of frames) controller.enqueue(enc.encode(f));
      controller.close();
    },
  });
  return new Response(stream, { status, headers: { "Content-Type": "text/event-stream" } });
}

export function restoreFetch() {
  vi.unstubAllGlobals();
}

/** authedMe is a convenient authenticated GET /me payload. By default a chosen path
 *  implies a fully-onboarded (completed) account; pass `completed` explicitly to model
 *  a mid-onboarding user (e.g. path chosen but budget not set). `enrollments` models the
 *  per-path Start state (F002) — default none (nothing started). */
export function authedMe(
  pathChosen: string | null = "dsa",
  completed: boolean = pathChosen !== null,
  enrollments: Array<{ path_slug: string; status: string; started_at: string }> = [],
) {
  return {
    account: {
      id: "acct-1",
      display_name: "Ada Lovelace",
      email: "ada@example.com",
      timezone: "UTC",
      study_budget: {},
      reminders: {},
      created_at: "2026-09-20T00:00:00Z",
    },
    onboarding: { path_chosen: pathChosen, budget_set: completed, key_added: false, completed },
    enrollments,
  };
}

/** enrolled models a started-path enrollment for authedMe's third arg. */
export function enrolled(slug = "dsa", startedAt = "2026-09-20T00:00:00Z") {
  return [{ path_slug: slug, status: "active", started_at: startedAt }];
}
