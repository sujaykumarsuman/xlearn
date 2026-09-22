import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, installFetchMock, restoreFetch } from "../test/fetchMock";

function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

interface Key {
  provider: string;
  masked_key: string;
  default_model: string;
  name: string;
  enabled: boolean;
  is_default: boolean;
}
function keysBody(keys: Key[]) {
  return { keys, connected: keys.length > 0, default_provider: keys.find((k) => k.is_default)?.provider ?? "" };
}

/** A settings fetch mock: /me (GET + PATCH capture) and /coach/key (GET, given a fixed coach state). */
function settingsMock(onPatch?: (body: unknown) => void, coach: Key[] = []) {
  return installFetchMock((url, init) => {
    if (url.endsWith("/api/me") && init?.method === "PATCH") {
      onPatch?.(init?.body ? JSON.parse(String(init.body)) : null);
      return { status: 200, body: authedMe("dsa") };
    }
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(coach) };
    return { status: 404 };
  });
}

/** The rendered "Your AI coach" card, for scoping queries away from the header switcher. */
function coachCard() {
  return screen.getByRole("heading", { name: /your ai coach/i }).closest("section")!;
}

describe("Settings screen", () => {
  afterEach(restoreFetch);

  it("renders the profile card (rail) + the budget, coach, and reminders sections", async () => {
    settingsMock();
    renderApp("/xlearn/settings");

    // Profile is the rail identity card (view mode): name + email on show.
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("ada@example.com")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /study budget/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /your ai coach/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /reminders/i })).toBeInTheDocument();
  });

  it("edits the profile via the Edit-profile form and saves via PATCH /me", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    fireEvent.click(await screen.findByRole("button", { name: /edit profile/i }));
    expect(screen.getByDisplayValue("ada@example.com")).toHaveAttribute("readonly"); // email read-only
    fireEvent.change(screen.getByDisplayValue("Ada Lovelace"), { target: { value: "Sujay Kumar" } });
    fireEvent.click(screen.getAllByRole("button", { name: /^save$/i })[0]!); // profile save (rail, first)

    await waitFor(() => expect(patched).toEqual({ display_name: "Sujay Kumar", timezone: "UTC" }));
  });

  it("cancels the Edit-profile form without saving", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    fireEvent.click(await screen.findByRole("button", { name: /edit profile/i }));
    fireEvent.change(screen.getByDisplayValue("Ada Lovelace"), { target: { value: "Discarded" } });
    fireEvent.click(screen.getByRole("button", { name: /^cancel$/i }));

    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(patched).toBeNull();
  });

  it("shows 'coach off' with both providers unconnected and gates connect on a key", async () => {
    settingsMock();
    renderApp("/xlearn/settings");

    await screen.findByRole("heading", { name: /your ai coach/i });
    const card = coachCard();
    expect((await within(card).findAllByText(/not connected/i)).length).toBe(2);
    expect(within(card).getByRole("button", { name: /connect anthropic/i })).toBeDisabled();
    fireEvent.change(within(card).getByPlaceholderText("sk-ant-…"), { target: { value: "sk-ant-secret-key-1234" } });
    expect(within(card).getByRole("button", { name: /connect anthropic/i })).toBeEnabled();
  });

  it("connects a provider via PUT /coach/key (provider + first model + name + key)", async () => {
    let putBody: unknown = null;
    let keys: Key[] = [];
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        keys = [{ provider: "anthropic", masked_key: "sk-ant-...1234", default_model: "claude-opus-5", name: "Opus 5", enabled: true, is_default: true }];
        return { status: 200, body: keysBody(keys) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(keys) };
      return { status: 404 };
    });
    renderApp("/xlearn/settings");

    await screen.findByRole("heading", { name: /your ai coach/i });
    const card = coachCard();
    fireEvent.change(await within(card).findByPlaceholderText("sk-ant-…"), { target: { value: "sk-ant-secret-key-1234" } });
    fireEvent.click(within(card).getByRole("button", { name: /connect anthropic/i }));

    // The first curated Anthropic model is the default; name defaults to its label.
    await waitFor(() => expect(putBody).toEqual({ provider: "anthropic", key: "sk-ant-secret-key-1234", default_model: "claude-opus-5", name: "Opus 5" }));
    expect(await within(coachCard()).findByText("sk-ant-...1234")).toBeInTheDocument();
  });

  it("switches a connected provider's model without a key (meta PUT)", async () => {
    let putBody: unknown = null;
    const saved: Key = { provider: "anthropic", masked_key: "sk-ant-...4a2f", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true };
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: keysBody([{ ...saved, default_model: "claude-opus-5" }]) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody([saved]) };
      return { status: 404 };
    });
    renderApp("/xlearn/settings");

    await screen.findByRole("heading", { name: /your ai coach/i });
    const card = coachCard();
    // Pick a different model pill, then Save (no key entered).
    fireEvent.click(await within(card).findByRole("button", { name: /^opus 5$/i }));
    fireEvent.click(within(card).getByRole("button", { name: /save anthropic/i }));
    await waitFor(() => expect(putBody).toEqual({ provider: "anthropic", default_model: "claude-opus-5", name: "Sonnet 5" }));
  });

  it("sets a provider as the default via PUT /coach/key", async () => {
    let putBody: unknown = null;
    const both: Key[] = [
      { provider: "anthropic", masked_key: "sk-ant-...4a2f", default_model: "claude-opus-5", name: "Opus 5", enabled: true, is_default: true },
      { provider: "openai", masked_key: "sk-...9f2c", default_model: "gpt-5.6-sol", name: "GPT-5.6 Sol", enabled: true, is_default: false },
    ];
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: keysBody(both) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(both) };
      return { status: 404 };
    });
    renderApp("/xlearn/settings");

    await screen.findByRole("heading", { name: /your ai coach/i });
    // Anthropic is default (badge); OpenAI offers "Set as default".
    fireEvent.click(await within(coachCard()).findByRole("button", { name: /set as default/i }));
    await waitFor(() => expect(putBody).toEqual({ provider: "openai", default: true }));
  });

  it("removes a provider via DELETE /coach/key?provider=", async () => {
    let deletedProvider = "";
    let keys: Key[] = [{ provider: "anthropic", masked_key: "sk-ant-...4a2f", default_model: "claude-opus-5", name: "Opus 5", enabled: true, is_default: true }];
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/key") && init?.method === "DELETE") {
        deletedProvider = new URL(url, "http://x").searchParams.get("provider") ?? "";
        keys = keys.filter((k) => k.provider !== deletedProvider);
        return { status: 204 };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(keys) };
      return { status: 404 };
    });
    renderApp("/xlearn/settings");

    await screen.findByRole("heading", { name: /your ai coach/i });
    fireEvent.click(await within(coachCard()).findByRole("button", { name: /^remove$/i }));
    await waitFor(() => expect(deletedProvider).toBe("anthropic"));
    expect(await screen.findByText(/coach off/i)).toBeInTheDocument();
  });

  it("steps the study budget and saves { weekday_minutes, weekend_band }", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    fireEvent.click(await screen.findByRole("button", { name: /more weekday minutes/i }));
    fireEvent.click(screen.getByRole("button", { name: /5h\+/i }));
    // Profile is in view mode (no Save), so budget is the first Save button.
    fireEvent.click(screen.getAllByRole("button", { name: /^save$/i })[0]!);

    await waitFor(() => expect(patched).toEqual({ study_budget: { weekday_minutes: 105, weekend_band: "5" } }));
  });

  it("saves reminder preferences via PATCH /me", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    fireEvent.click(await screen.findByRole("switch", { name: /daily study reminder/i }));
    fireEvent.click(screen.getAllByRole("button", { name: /^save$/i })[1]!);

    await waitFor(() =>
      expect(patched).toEqual({
        reminders: { daily_reminder_on: false, daily_reminder_time: "20:00", revision_due_alerts_on: true },
      }),
    );
  });
});
