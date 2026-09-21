import { vi } from "vitest";

/** A minimal route handler: returns the status + JSON body for a request. */
export type RouteHandler = (url: string, init?: RequestInit) => { status: number; body?: unknown };

/** installFetchMock stubs global fetch with a route handler. Returns the vi mock
 *  so tests can assert on calls. Pair with restoreFetch() in afterEach. */
export function installFetchMock(handler: RouteHandler) {
  const fn = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = typeof input === "string" ? input : input.toString();
    const { status, body } = handler(url, init);
    return new Response(body === undefined ? null : JSON.stringify(body), {
      status,
      headers: { "Content-Type": "application/json" },
    });
  });
  vi.stubGlobal("fetch", fn);
  return fn;
}

export function restoreFetch() {
  vi.unstubAllGlobals();
}

/** authedMe is a convenient authenticated GET /me payload. By default a chosen path
 *  implies a fully-onboarded (completed) account; pass `completed` explicitly to model
 *  a mid-onboarding user (e.g. path chosen but budget not set). */
export function authedMe(pathChosen: string | null = "dsa", completed: boolean = pathChosen !== null) {
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
  };
}
