import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { DSA_PATH, ZZ_FIXTURE_PATH, ZZ_SOON_PATH, catalog } from "../test/courses";
import { authedMe, installFetchMock, restoreFetch, sseResponse, type RouteHandler } from "../test/fetchMock";

function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

const enabledKey = { keys: [{ provider: "openai", masked_key: "sk-...1234", default_model: "gpt-4o-mini", enabled: true }], connected: true };
const noKey = { keys: [], connected: false };

/** coachMock wires /me + a coach-key state + a thread + a chat handler. */
function coachMock(opts: {
  key?: { keys: unknown[]; connected: boolean };
  thread?: unknown[];
  chat?: (init?: RequestInit) => Response | { status: number; body?: unknown };
}): void {
  const handler: RouteHandler = (url, init) => {
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.endsWith("/api/coach/key")) return { status: 200, body: opts.key ?? noKey };
    if (url.includes("/api/coach/thread")) return { status: 200, body: { messages: opts.thread ?? [] } };
    if (url.endsWith("/api/coach/chat")) return opts.chat ? opts.chat(init) : { status: 200, body: {} };
    return { status: 404 };
  };
  installFetchMock(handler);
}

describe("Coach panel", () => {
  afterEach(restoreFetch);

  it("shows the FAB and opens the panel", async () => {
    coachMock({ key: enabledKey });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    expect(await screen.findByRole("complementary", { name: /ai coach/i })).toBeInTheDocument();
  });

  it("shows the empty state and routes to Settings when there is no key", async () => {
    coachMock({ key: noKey });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    expect(await screen.findByText(/your ai coach is off/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /open settings/i }));
    expect(await screen.findByRole("heading", { level: 1, name: "Settings" })).toBeInTheDocument();
  });

  it("streams a reply into the transcript and posts the page context", async () => {
    let chatBody: { context?: string; message?: string } = {};
    coachMock({
      key: enabledKey,
      chat: (init) => {
        chatBody = init?.body ? JSON.parse(String(init.body)) : {};
        return sseResponse([`data: {"delta":"Hi "}\n\n`, `data: {"delta":"there"}\n\n`, `data: {"done":true}\n\n`]);
      },
    });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));

    const input = await screen.findByLabelText(/message coach/i);
    fireEvent.change(input, { target: { value: "how am I doing?" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));

    expect(await screen.findByText("Hi there")).toBeInTheDocument();
    expect(screen.getByText("how am I doing?")).toBeInTheDocument();
    // Course-scoped contexts carry their course (m1-03): the thread key is per course.
    await waitFor(() => expect(chatBody.context).toBe("dsa:dashboard"));
    expect(chatBody.message).toBe("how am I doing?");
  });

  it("loads the existing thread history on open", async () => {
    coachMock({ key: enabledKey, thread: [{ role: "assistant", content: "We spoke about two pointers earlier." }] });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    expect(await screen.findByText(/two pointers earlier/i)).toBeInTheDocument();
  });

  it("routes to Settings when the provider rejects the key mid-stream", async () => {
    coachMock({
      key: enabledKey,
      chat: () => sseResponse([`data: {"error":"provider_auth","message":"rejected"}\n\n`]),
    });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    const input = await screen.findByLabelText(/message coach/i);
    fireEvent.change(input, { target: { value: "hello" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    expect(await screen.findByRole("heading", { level: 1, name: "Settings" })).toBeInTheDocument();
  });

  /** sendAndFail sends a turn that the provider refuses with the given `reason`, and
   *  returns once the panel has rendered its line for it. */
  async function sendAndFail(reason: string, opts: { status?: number; key?: { keys: unknown[]; connected: boolean } } = {}) {
    coachMock({
      key: opts.key ?? enabledKey,
      chat: () => ({
        status: opts.status ?? 429,
        body: { error: { code: "provider_limited", message: "your provider account is out of credit or limited — top up and retry", reason } },
      }),
    });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    fireEvent.change(await screen.findByLabelText(/message coach/i), { target: { value: "hint please" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    return screen.findByRole("alert");
  }

  it.each([
    // Quota and a rate limit share one line: both mean "the account, not the key".
    ["quota", "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry."],
    ["rate_limit", "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry."],
    ["region", "Your provider doesn't serve this region."],
  ])("reason=%s shows its own line and keeps the coach on", async (reason, line) => {
    const alert = await sendAndFail(reason);
    expect(alert).toHaveTextContent(line);
    // No bounce to Settings: the key is fine, so the panel stays usable.
    expect(screen.queryByRole("heading", { level: 1, name: "Settings" })).not.toBeInTheDocument();
    expect(screen.getByLabelText(/message coach/i)).toBeInTheDocument();
  });

  it("reason=model_access offers the catalog switcher instead of blaming the key", async () => {
    const alert = await sendAndFail("model_access");
    expect(alert).toHaveTextContent("This key can't use gpt-4o-mini. Pick another model.");
    // The list opens with the affordance, and the failing model stays listed and marked.
    const list = await screen.findByRole("listbox", { name: /coach model/i });
    expect(within(list).getByText("This key can't use it")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /pick another model/i })).toHaveAttribute("aria-expanded", "true");
    // The key is untouched — a model error is never a disable.
    expect(screen.queryByRole("heading", { level: 1, name: "Settings" })).not.toBeInTheDocument();
  });

  it("falls back to the generic line when the server sends no reason (a pre-m1-10 coach)", async () => {
    coachMock({ key: enabledKey, chat: () => ({ status: 502, body: { error: { code: "provider_error", message: "upstream" } } }) });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    fireEvent.change(await screen.findByLabelText(/message coach/i), { target: { value: "hint please" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    expect(await screen.findByText(/The coach couldn’t reply just now/i)).toBeInTheDocument();
  });

  it("shows F15b's copy when the key itself has been turned off", async () => {
    coachMock({ key: { keys: [{ provider: "openai", masked_key: "sk-...1234", default_model: "gpt-4o-mini", enabled: false }], connected: true } });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    expect(await screen.findByText("Your coach key is turned off")).toBeInTheDocument();
    expect(screen.getByText("Re-enable your provider key in Settings to turn the coach back on.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add your key in settings/i })).toBeInTheDocument();
  });

  it("reflects the current page in the context chip", async () => {
    coachMock({ key: enabledKey });
    renderApp("/xlearn/dsa/concept/sliding-window");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    expect(await screen.findByText(/Concept — Sliding Window/i)).toBeInTheDocument();
  });
});

describe("Coach page context (m1-03: <course>:<ctx>)", () => {
  afterEach(restoreFetch);

  /** threadContextAt opens the coach at a path and returns the context its thread read
   *  asked for (GET /coach/thread?context=…), plus the chip text. */
  async function threadContextAt(path: string, paths?: unknown): Promise<{ context: string; chip: string }> {
    let context = "";
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (paths && url.endsWith("/api/paths")) return { status: 200, body: paths };
      if (url.endsWith("/api/coach/key")) return { status: 200, body: enabledKey };
      if (url.includes("/api/coach/thread")) {
        context = new URL(url, "http://x").searchParams.get("context") ?? "";
        return { status: 200, body: { messages: [] } };
      }
      return { status: 404 };
    });
    const { unmount } = renderApp(path);
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    await waitFor(() => expect(context).not.toBe(""));
    const chip = document.querySelector(".xl-coach__ctx")?.textContent ?? "";
    unmount();
    return { context, chip };
  }

  it.each([
    ["/xlearn/dsa", "dsa:roadmap", "Roadmap"],
    ["/xlearn/dsa/dashboard", "dsa:dashboard", "Today"],
    ["/xlearn/dsa/revision", "dsa:revision", "Revision"],
    ["/xlearn/dsa/mistakes", "dsa:mistakes", "Mistakes"],
    ["/xlearn/dsa/mock", "dsa:mock", "Mock interview"],
    ["/xlearn/dsa/progress", "dsa:progress", "Progress"],
    ["/xlearn/dsa/week/3", "dsa:week:3", "Week 3"],
    ["/xlearn/dsa/concept/two-pointers", "dsa:concept:two-pointers", "Concept — Two Pointers"],
    // Problems are global ids: unchanged.
    ["/xlearn/dsa/problem/16", "problem:16", "Problem 16"],
    // Account-wide contexts are unchanged.
    ["/xlearn", "catalog", "Catalog"],
    ["/xlearn/settings", "settings", "Settings"],
    // Not a course (NotFound in the plain frame): general.
    ["/xlearn/nope", "general", "xLearn"],
    ["/xlearn/dsa/bogus", "general", "xLearn"],
  ])("%s → %s", async (path, want, chip) => {
    const got = await threadContextAt(path);
    expect(got.context).toBe(want);
    expect(got.chip).toContain(`Context · ${chip}`);
  });

  it("gives a second course its own keys (week 3 of two courses are separate threads)", async () => {
    const both = catalog(DSA_PATH, ZZ_FIXTURE_PATH);
    expect((await threadContextAt("/xlearn/zz-fixture/week/3", both)).context).toBe("zz-fixture:week:3");
    expect((await threadContextAt("/xlearn/dsa/week/3", both)).context).toBe("dsa:week:3");
    expect((await threadContextAt("/xlearn/zz-fixture/dashboard", both)).context).toBe("zz-fixture:dashboard");
  });

  it("uses the general context on a coming-soon course's teaser", async () => {
    const got = await threadContextAt("/xlearn/zz-soon/week/3", catalog(DSA_PATH, ZZ_SOON_PATH));
    expect(got.context).toBe("general");
  });
});
