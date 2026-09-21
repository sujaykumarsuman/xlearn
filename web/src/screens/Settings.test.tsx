import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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

/** A settings fetch mock: /me (GET), /coach/key, and a PATCH /me capture. */
function settingsMock(onPatch?: (body: unknown) => void, coach: { keys: unknown[]; connected: boolean } = { keys: [], connected: false }) {
  return installFetchMock((url, init) => {
    if (url.endsWith("/api/me") && init?.method === "PATCH") {
      const body = init?.body ? JSON.parse(String(init.body)) : null;
      onPatch?.(body);
      const me = authedMe("dsa");
      return { status: 200, body: me };
    }
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.endsWith("/api/coach/key")) return { status: 200, body: coach };
    return { status: 404 };
  });
}

describe("Settings screen", () => {
  afterEach(restoreFetch);

  it("renders the profile, budget, reminders, and API-keys sections", async () => {
    settingsMock();
    renderApp("/xlearn/settings");

    expect(await screen.findByRole("heading", { name: /^profile$/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /study budget/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /api keys/i })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /reminders/i })).toBeInTheDocument();
    // Email is read-only.
    expect(screen.getByLabelText(/email/i)).toHaveAttribute("readonly");
  });

  it("shows the 'no key — coach off' empty state when there is no key", async () => {
    settingsMock();
    renderApp("/xlearn/settings");
    expect(await screen.findByText(/no key — coach off/i)).toBeInTheDocument();
    // The store/delete are shell-only: the add form's "Add key" is present but disabled.
    fireEvent.click(await screen.findByRole("button", { name: /add a provider/i }));
    expect(screen.getByRole("button", { name: /add key/i })).toBeDisabled();
  });

  it("shows a Connected badge when coach reports an enabled key", async () => {
    settingsMock(undefined, {
      keys: [{ provider: "Anthropic", masked_key: "sk-ant-****4a2f", default_model: "claude-sonnet-5", enabled: true, tested: true }],
      connected: true,
    });
    renderApp("/xlearn/settings");
    // "Connected" appears both as the header status and the per-key badge.
    expect((await screen.findAllByText(/^connected$/i)).length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("sk-ant-****4a2f")).toBeInTheDocument();
    expect(screen.getByText("claude-sonnet-5")).toBeInTheDocument();
  });

  it("saves profile edits via PATCH /me", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    const name = await screen.findByLabelText(/full name/i);
    fireEvent.change(name, { target: { value: "Sujay Kumar" } });
    fireEvent.click(screen.getAllByRole("button", { name: /^save$/i })[0]!);

    await waitFor(() => expect(patched).toEqual({ display_name: "Sujay Kumar", timezone: "UTC" }));
    expect(await screen.findByText(/saved/i)).toBeInTheDocument();
  });

  it("steps the study budget and saves { weekday_minutes, weekend_band }", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    fireEvent.click(await screen.findByRole("button", { name: /more weekday minutes/i })); // 90 → 105
    fireEvent.click(screen.getByRole("button", { name: /5h\+/i })); // weekend band → "5"
    fireEvent.click(screen.getAllByRole("button", { name: /^save$/i })[1]!);

    await waitFor(() => expect(patched).toEqual({ study_budget: { weekday_minutes: 105, weekend_band: "5" } }));
  });

  it("saves reminder preferences via PATCH /me", async () => {
    let patched: unknown = null;
    settingsMock((b) => (patched = b));
    renderApp("/xlearn/settings");

    // Turn the daily reminder off, then save the reminders section.
    fireEvent.click(await screen.findByRole("switch", { name: /daily study reminder/i }));
    fireEvent.click(screen.getAllByRole("button", { name: /^save$/i })[2]!);

    await waitFor(() =>
      expect(patched).toEqual({
        reminders: { daily_reminder_on: false, daily_reminder_time: "20:00", revision_due_alerts_on: true },
      }),
    );
  });
});
