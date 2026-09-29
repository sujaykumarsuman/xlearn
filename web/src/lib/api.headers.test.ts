// The gateway refuses a mutating /api call that isn't application/json (415
// unsupported_media_type) — the cross-site write guard (m1-04; ADR-0033 §9). These tests
// fail if either client path that writes to the BFF stops sending the JSON content type:
// apiFetch (every screen's writes) and streamCoachChat (the coach SSE relay, a raw fetch).
import { afterEach, describe, expect, it, vi } from "vitest";
import { apiFetch } from "./api";
import { streamCoachChat } from "./settings";
import { restoreFetch, sseResponse } from "../test/fetchMock";

/** stubFetch records every call's method + normalised headers and answers `respond()`. */
function stubFetch(respond: () => Response) {
  const calls: { url: string; method: string; headers: Headers }[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      calls.push({
        url: typeof input === "string" ? input : input.toString(),
        method: (init?.method ?? "GET").toUpperCase(),
        headers: new Headers(init?.headers),
      });
      return respond();
    }),
  );
  return calls;
}

const json = () => new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } });

describe("apiFetch request headers", () => {
  afterEach(restoreFetch);

  it.each([
    ["POST with a body", { method: "POST", body: JSON.stringify({ a: 1 }) }],
    ["POST without a body", { method: "POST" }],
    ["PUT", { method: "PUT", body: JSON.stringify({ provider: "anthropic" }) }],
    ["PATCH", { method: "PATCH", body: JSON.stringify({ display_name: "x" }) }],
    ["DELETE", { method: "DELETE" }],
    ["lower-case method", { method: "post" }],
  ] as const)("sends Content-Type: application/json on %s", async (_name, init) => {
    const calls = stubFetch(json);
    await apiFetch("/x", init);
    expect(calls).toHaveLength(1);
    expect(calls[0]!.headers.get("Content-Type")).toBe("application/json");
    expect(calls[0]!.headers.get("Accept")).toBe("application/json");
  });

  it("does not double a Content-Type the caller already set (any casing)", async () => {
    const variants: Record<string, string>[] = [{ "Content-Type": "application/json" }, { "content-type": "application/json" }];
    for (const headers of variants) {
      const calls = stubFetch(json);
      await apiFetch("/x", { method: "POST", headers, body: "{}" });
      expect(calls[0]!.headers.get("Content-Type")).toBe("application/json");
      restoreFetch();
    }
  });

  it("keeps a caller's other headers", async () => {
    const calls = stubFetch(json);
    await apiFetch("/x", { method: "POST", headers: new Headers({ "X-Trace": "t1" }) });
    expect(calls[0]!.headers.get("X-Trace")).toBe("t1");
    expect(calls[0]!.headers.get("Content-Type")).toBe("application/json");
  });

  it("sends no Content-Type on GET or HEAD (reads carry no body)", async () => {
    for (const init of [undefined, { method: "GET" }, { method: "HEAD" }]) {
      const calls = stubFetch(json);
      await apiFetch("/x", init).catch(() => undefined);
      expect(calls[0]!.headers.has("Content-Type")).toBe(false);
      expect(calls[0]!.headers.get("Accept")).toBe("application/json");
      restoreFetch();
    }
  });
});

describe("streamCoachChat request headers", () => {
  afterEach(restoreFetch);

  it("POSTs the chat turn as application/json (and asks for an event stream)", async () => {
    const calls = stubFetch(() => sseResponse(['data: {"delta":"hi"}\n\n', 'data: {"done":true}\n\n']));
    const deltas: string[] = [];
    await streamCoachChat({ context: "dashboard", message: "hi" }, (d) => deltas.push(d));
    expect(calls).toHaveLength(1);
    expect(calls[0]!.url).toMatch(/\/api\/v1\/coach\/chat$/);
    expect(calls[0]!.method).toBe("POST");
    expect(calls[0]!.headers.get("Content-Type")).toBe("application/json");
    expect(calls[0]!.headers.get("Accept")).toBe("text/event-stream");
    expect(deltas).toEqual(["hi"]);
  });
});
