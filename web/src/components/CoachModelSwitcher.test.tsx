import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, installFetchMock, restoreFetch } from "../test/fetchMock";

/** The top-bar switcher lives in the app shell, so these tests drive it through a real
 *  route rather than mounting it bare — that is also what proves it stays silent until a
 *  coach is actually connected. */
function renderApp(path = "/xlearn/dsa/dashboard") {
  const router = createMemoryRouter(routes, { initialEntries: [path], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

/** The server catalog, trimmed to what these tests assert on: a recommended model that is
 *  also an interview brain, a cheap one priced in cents, and a Covered Model. */
const MODELS = {
  as_of: "2026-10-01",
  providers: ["anthropic", "openai"],
  models: [
    {
      id: "claude-sonnet-5",
      provider: "anthropic",
      label: "Sonnet 5",
      capabilities: ["chat", "interview_brain"],
      price: { input_micros_per_mtok: 2_000_000, output_micros_per_mtok: 10_000_000 },
      as_of: "2026-10-01",
      recommended: true,
      covered_model: false,
    },
    {
      id: "claude-opus-5-5",
      provider: "anthropic",
      label: "Opus 5.5",
      capabilities: ["chat", "interview_brain", "voice_shell"],
      price: { input_micros_per_mtok: 4_000_000, output_micros_per_mtok: 20_000_000 },
      as_of: "2026-10-01",
      recommended: false,
      covered_model: true,
    },
    {
      id: "gpt-6-luna",
      provider: "openai",
      label: "GPT-6 Luna",
      capabilities: ["chat"],
      price: { input_micros_per_mtok: 100_000, output_micros_per_mtok: 500_000 },
      as_of: "2026-10-01",
      recommended: true,
      covered_model: false,
    },
  ],
  defaults: { anthropic: "claude-sonnet-5", openai: "gpt-6-luna" },
};

interface Key {
  provider: string;
  masked_key: string;
  default_model: string;
  name: string;
  enabled: boolean;
  is_default: boolean;
}
const ANTHROPIC: Key = { provider: "anthropic", masked_key: "sk-ant-…4f2a", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true };
const OPENAI: Key = { provider: "openai", masked_key: "sk-…9c1e", default_model: "gpt-6-luna", name: "GPT-6 Luna", enabled: true, is_default: false };

/** switcherMock serves /me, the catalog and a coach-key state, and records PUT bodies.
 *  `catalog` false makes GET /coach/models unavailable. */
function switcherMock(keys: Key[], opts: { catalog?: boolean; coachModel?: string } = {}) {
  const puts: unknown[] = [];
  const coach = keys.find((k) => k.is_default) ?? keys[0];
  const body = () => ({
    keys,
    connected: keys.length > 0,
    default_provider: coach?.provider ?? "",
    defaults: { coach: coach ? { provider: coach.provider, model: opts.coachModel ?? coach.default_model } : null, interview: null },
    usage_month: { messages: 0, input_tokens: 0, output_tokens: 0, est_cost_micros: 0, has_unknown_cost: false },
  });
  installFetchMock((url, init) => {
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.includes("/api/coach/models")) return opts.catalog === false ? { status: 503, body: { error: { code: "unavailable" } } } : { status: 200, body: MODELS };
    if (url.includes("/api/coach/key") && init?.method === "PUT") {
      puts.push(JSON.parse(String(init.body)));
      return { status: 200, body: body() };
    }
    if (url.includes("/api/coach/key")) return { status: 200, body: body() };
    return { status: 404 };
  });
  return puts;
}

/** openMenu clicks the model button and returns the open listbox. */
async function openMenu() {
  fireEvent.click(await screen.findByRole("button", { name: /coach model/i }));
  return screen.getByRole("listbox", { name: /coach model/i });
}

describe("CoachModelSwitcher (AB01 F12)", () => {
  afterEach(restoreFetch);

  it("renders nothing until a coach is connected", async () => {
    switcherMock([]);
    renderApp();
    await screen.findByRole("heading", { level: 1, name: "Today" });
    expect(screen.queryByRole("button", { name: /coach model/i })).not.toBeInTheDocument();
  });

  it("lists the active provider's catalog models with tags, dated prices and the badge", async () => {
    switcherMock([ANTHROPIC]);
    renderApp();
    const list = await openMenu();

    expect(within(list).getByText("Anthropic · models")).toBeInTheDocument();
    expect(within(list).getByText("Prices per MTok, as of Oct 1, 2026")).toBeInTheDocument();

    const options = within(list).getAllByRole("option");
    expect(options.map((o) => o.textContent)).toEqual([
      "Sonnet 5Recommended" + "Interview" + "$2 in · $10 out",
      "Opus 5.5InterviewVoice$4 in · $20 outYour provider keeps these chats 30 days",
      " Custom model id…", // the leading space is the row's plus icon
    ]);
    // The model in use is the selected option.
    expect(options[0]).toHaveAttribute("aria-selected", "true");
    expect(options[1]).toHaveAttribute("aria-selected", "false");
    // The other provider's models never leak into the active list.
    expect(within(list).queryByText("GPT-6 Luna")).not.toBeInTheDocument();
  });

  it("prices sub-dollar models to the cent", async () => {
    switcherMock([OPENAI]);
    renderApp();
    const list = await openMenu();
    expect(within(list).getByText("$0.10 in · $0.50 out")).toBeInTheDocument();
  });

  it("picking a model sets the coach feature default", async () => {
    const puts = switcherMock([ANTHROPIC]);
    renderApp();
    const list = await openMenu();
    fireEvent.click(within(list).getByRole("option", { name: /opus 5\.5/i }));

    await waitFor(() => expect(puts).toEqual([{ provider: "anthropic", default: true, feature: "coach", default_model: "claude-opus-5-5" }]));
    // The list closes on a pick.
    expect(screen.queryByRole("listbox", { name: /coach model/i })).not.toBeInTheDocument();
  });

  it("switching the provider pill re-points the coach default at that key", async () => {
    const puts = switcherMock([ANTHROPIC, OPENAI]);
    renderApp();
    const pills = await screen.findByRole("group", { name: /coach provider/i });
    fireEvent.click(within(pills).getByRole("button", { name: "OpenAI" }));
    await waitFor(() => expect(puts).toEqual([{ provider: "openai", default: true, feature: "coach" }]));
  });

  it("accepts a custom model id and rejects a URL typed as one", async () => {
    const puts = switcherMock([ANTHROPIC]);
    renderApp();
    const list = await openMenu();
    fireEvent.click(within(list).getByRole("option", { name: /custom model id/i }));

    const field = within(list).getByLabelText("Model id");
    fireEvent.change(field, { target: { value: "https://proxy.example.com/v1" } });
    expect(field).toHaveAttribute("aria-invalid", "true");
    expect(within(list).getByText("That isn't a model id. Use letters, digits and . _ : - (no URLs).")).toBeInTheDocument();
    expect(within(list).getByRole("button", { name: /^use$/i })).toBeDisabled();

    // A fine-tune id is accepted, and flagged as having no published price.
    fireEvent.change(field, { target: { value: "ft:claude-sonnet-5:personal:coach" } });
    expect(field).not.toHaveAttribute("aria-invalid");
    expect(within(list).getByText("cost unknown")).toBeInTheDocument();
    expect(within(list).getByText("Not in the catalog, so this month's estimate leaves it out.")).toBeInTheDocument();
    fireEvent.click(within(list).getByRole("button", { name: /^use$/i }));

    await waitFor(() => expect(puts).toEqual([{ provider: "anthropic", default: true, feature: "coach", default_model: "ft:claude-sonnet-5:personal:coach" }]));
  });

  it("offers no base-URL field anywhere in the menu", async () => {
    switcherMock([ANTHROPIC]);
    renderApp();
    const list = await openMenu();
    fireEvent.click(within(list).getByRole("option", { name: /custom model id/i }));
    // A learner picks a model id, never an endpoint (the API 400s a base_url outright).
    expect(within(list).getAllByRole("textbox").map((i) => i.getAttribute("aria-label") ?? i.id)).toHaveLength(1);
    expect(within(list).queryByLabelText(/base url/i)).not.toBeInTheDocument();
  });

  it("shows only the model in use when the catalog can't be read", async () => {
    switcherMock([ANTHROPIC], { catalog: false });
    renderApp();
    const list = await openMenu();

    const options = within(list).getAllByRole("option");
    expect(options.map((o) => o.textContent)).toEqual(["claude-sonnet-5cost unknown", " Custom model id…"]);
    expect(within(list).queryByText(/Prices per MTok/)).not.toBeInTheDocument();
  });

  it("keeps a custom model in use visible beside the catalog's own entries", async () => {
    switcherMock([ANTHROPIC], { coachModel: "ft:claude-sonnet-5:personal:coach" });
    renderApp();
    // The button reads the id itself — there is no label to resolve for a custom model.
    expect(await screen.findByRole("button", { name: /coach model/i })).toHaveTextContent("ft:claude-sonnet-5:personal:coach");
    const list = await openMenu();
    const selected = within(list).getByRole("option", { name: /ft:claude-sonnet-5:personal:coach/ });
    expect(selected).toHaveAttribute("aria-selected", "true");
    expect(within(list).getByRole("option", { name: /sonnet 5/i })).toHaveAttribute("aria-selected", "false");
  });

  it("closes on Escape", async () => {
    switcherMock([ANTHROPIC]);
    renderApp();
    await openMenu();
    fireEvent.keyDown(document, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("listbox", { name: /coach model/i })).not.toBeInTheDocument());
  });
});
