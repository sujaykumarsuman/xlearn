import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
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
    await waitFor(() => expect(chatBody.context).toBe("dashboard"));
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

  it("reflects the current page in the context chip", async () => {
    coachMock({ key: enabledKey });
    renderApp("/xlearn/dsa/concept/sliding-window");
    fireEvent.click(await screen.findByRole("button", { name: /open ai coach/i }));
    expect(await screen.findByText(/Concept — Sliding Window/i)).toBeInTheDocument();
  });
});
