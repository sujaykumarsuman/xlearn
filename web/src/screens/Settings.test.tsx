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
/** Per-feature defaults + month-to-date usage, as m1-10's GET /coach/key returns them. */
interface CoachState {
  defaults?: { coach?: { provider: string; model: string } | null; interview?: { provider: string; model: string } | null };
  usage?: { messages: number; input_tokens: number; output_tokens: number; est_cost_micros: number; has_unknown_cost: boolean };
}
function keysBody(keys: Key[], state: CoachState = {}) {
  const coach = state.defaults?.coach ?? defaultFromKeys(keys);
  return {
    keys,
    connected: keys.length > 0,
    default_provider: coach?.provider ?? "",
    defaults: { coach: coach ?? null, interview: state.defaults?.interview ?? null },
    usage_month: state.usage ?? { messages: 0, input_tokens: 0, output_tokens: 0, est_cost_micros: 0, has_unknown_cost: false },
  };
}
function defaultFromKeys(keys: Key[]) {
  const k = keys.find((x) => x.is_default) ?? keys[0];
  return k ? { provider: k.provider, model: k.default_model } : null;
}

/** The server model catalog as GET /coach/models serves it (a trimmed stand-in: two
 *  Anthropic models, one of them interview-capable, and two OpenAI ones). */
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
      id: "claude-opus-4-8",
      provider: "anthropic",
      label: "Opus 4.8",
      capabilities: ["chat"],
      price: { input_micros_per_mtok: 5_000_000, output_micros_per_mtok: 25_000_000 },
      as_of: "2026-10-01",
      recommended: false,
      covered_model: false,
    },
    {
      id: "gpt-6-sol",
      provider: "openai",
      label: "GPT-6 Sol",
      capabilities: ["chat", "interview_brain"],
      price: { input_micros_per_mtok: 2_000_000, output_micros_per_mtok: 10_000_000 },
      as_of: "2026-10-01",
      recommended: false,
      covered_model: false,
    },
    {
      id: "gpt-5.6-luna",
      provider: "openai",
      label: "GPT-5.6 Luna",
      capabilities: ["chat"],
      price: { input_micros_per_mtok: 200_000, output_micros_per_mtok: 1_200_000 },
      as_of: "2026-10-01",
      recommended: true,
      covered_model: false,
    },
  ],
  defaults: { anthropic: "claude-sonnet-5", openai: "gpt-5.6-luna" },
};

/** A settings fetch mock: /me (GET + PATCH capture), /coach/key (GET, given a fixed coach
 *  state) and the model catalog. */
function settingsMock(onPatch?: (body: unknown) => void, coach: Key[] = [], state?: CoachState) {
  return installFetchMock((url, init) => {
    if (url.endsWith("/api/me") && init?.method === "PATCH") {
      onPatch?.(init?.body ? JSON.parse(String(init.body)) : null);
      return { status: 200, body: authedMe("dsa") };
    }
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.includes("/api/coach/models")) return { status: 200, body: MODELS };
    if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(coach, state) };
    return { status: 404 };
  });
}

/** openCoachTab renders Settings on the coach tab and resolves once the card is up. */
async function openCoachTab() {
  renderApp("/xlearn/settings?tab=coach");
  await screen.findByRole("heading", { name: /your ai coach/i });
}

/** The rendered "Your AI coach" card, for scoping queries away from the header switcher. */
function coachCard() {
  return screen.getByRole("heading", { name: /your ai coach/i }).closest("section")!;
}

describe("Settings screen", () => {
  afterEach(restoreFetch);

  it("renders the profile card + section tabs, and switches the panel on tab click", async () => {
    settingsMock();
    renderApp("/xlearn/settings");

    // Profile is the rail identity card (view mode): name + email on show.
    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("ada@example.com")).toBeInTheDocument();

    // The rail is a tablist; Study budget is selected by default and only its panel renders.
    expect(screen.getByRole("tab", { name: /study budget/i })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("heading", { name: /study budget/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /your ai coach/i })).not.toBeInTheDocument();

    // Switching tabs swaps the panel (this is what a scroll-spy rail couldn't do reliably).
    fireEvent.click(screen.getByRole("tab", { name: /your ai coach/i }));
    expect(await screen.findByRole("heading", { name: /your ai coach/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /study budget/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: /reminders/i }));
    expect(await screen.findByRole("heading", { name: /reminders/i })).toBeInTheDocument();
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

    fireEvent.click(await screen.findByRole("tab", { name: /your ai coach/i }));
    await screen.findByRole("heading", { name: /your ai coach/i });
    const card = coachCard();
    expect((await within(card).findAllByText(/not connected/i)).length).toBe(2);
    expect(within(card).getByRole("button", { name: /connect anthropic/i })).toBeDisabled();
    fireEvent.change(within(card).getByPlaceholderText("sk-ant-…"), { target: { value: "sk-ant-secret-key-1234" } });
    expect(within(card).getByRole("button", { name: /connect anthropic/i })).toBeEnabled();
  });

  it("connects a provider via PUT /coach/key (provider + key; the server picks the model)", async () => {
    let putBody: unknown = null;
    let keys: Key[] = [];
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/models")) return { status: 200, body: MODELS };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        keys = [{ provider: "anthropic", masked_key: "sk-ant-...1234", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true }];
        return { status: 200, body: keysBody(keys) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(keys) };
      return { status: 404 };
    });
    await openCoachTab();

    const card = coachCard();
    fireEvent.change(await within(card).findByLabelText("Anthropic API key"), { target: { value: "sk-ant-secret-key-1234" } });
    fireEvent.click(within(card).getByRole("button", { name: /connect anthropic/i }));

    // A first connect names no model: the server applies the catalog default, so the SPA
    // can't pin a stale id into a brand-new key.
    await waitFor(() => expect(putBody).toEqual({ provider: "anthropic", key: "sk-ant-secret-key-1234" }));
    expect(await within(coachCard()).findByDisplayValue("sk-ant-...1234")).toBeInTheDocument();
  });

  it("replacing a key keeps the model the key already had", async () => {
    let putBody: unknown = null;
    const saved: Key = { provider: "anthropic", masked_key: "sk-ant-...4a2f", default_model: "claude-opus-4-8", name: "Opus 4.8", enabled: true, is_default: true };
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/models")) return { status: 200, body: MODELS };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: keysBody([saved]) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody([saved]) };
      return { status: 404 };
    });
    await openCoachTab();

    const card = coachCard();
    fireEvent.click(await within(card).findByRole("button", { name: /^update$/i }));
    fireEvent.change(within(card).getByLabelText("Anthropic API key"), { target: { value: "sk-ant-rotated-9999" } });
    fireEvent.click(within(card).getByRole("button", { name: /save anthropic/i }));
    // Without default_model the server would reset the model to the catalog default —
    // silently undoing the learner's pick on every key rotation.
    await waitFor(() => expect(putBody).toEqual({ provider: "anthropic", key: "sk-ant-rotated-9999", default_model: "claude-opus-4-8" }));
  });

  it("shows each feature's default model, the month-to-date line and the key role chips", async () => {
    const keys: Key[] = [
      { provider: "anthropic", masked_key: "sk-ant-…4f2a", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true },
      { provider: "openai", masked_key: "sk-…9c1e", default_model: "gpt-6-sol", name: "GPT-6 Sol", enabled: true, is_default: false },
    ];
    settingsMock(undefined, keys, {
      defaults: { coach: { provider: "anthropic", model: "claude-sonnet-5" }, interview: { provider: "openai", model: "gpt-6-sol" } },
      usage: { messages: 212, input_tokens: 1_000_000, output_tokens: 400_000, est_cost_micros: 3_100_000, has_unknown_cost: false },
    });
    await openCoachTab();
    const card = coachCard();

    // Coach row: label from the catalog + the id + its provider. Interview row carries the
    // capability parenthetical.
    expect(await within(card).findByText("Sonnet 5")).toBeInTheDocument();
    expect(within(card).getByText("claude-sonnet-5")).toBeInTheDocument();
    expect(within(card).getByText("GPT-6 Sol")).toBeInTheDocument();
    expect(within(card).getByText("(needs an interview-capable model)")).toBeInTheDocument();
    // Both rows are set, so both read Change (never Choose).
    expect(within(card).getAllByRole("button", { name: /^change$/i })).toHaveLength(2);

    expect(within(card).getByText("212 messages · 1.4 M tokens · ≈ $3.10 (estimate)")).toBeInTheDocument();
    expect(within(card).queryByText(/custom models not estimated/)).not.toBeInTheDocument();

    expect(within(card).getByText("Coach default")).toBeInTheDocument();
    expect(within(card).getByText("Interview default")).toBeInTheDocument();
  });

  it("reads 'Not set' with a Choose affordance when the interview default is unset", async () => {
    settingsMock(undefined, [{ provider: "anthropic", masked_key: "sk-ant-…4f2a", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true }], {
      usage: { messages: 96, input_tokens: 400_000, output_tokens: 200_000, est_cost_micros: 1_200_000, has_unknown_cost: true },
    });
    await openCoachTab();
    const card = coachCard();

    expect(await within(card).findByText("Not set")).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: /^choose$/i })).toBeInTheDocument();
    // has_unknown_cost appends the note outside the figures, so the number is never read
    // as a total.
    expect(within(card).getByText("96 messages · 0.6 M tokens · ≈ $1.20 (estimate)")).toBeInTheDocument();
    expect(within(card).getByText(/· custom models not estimated/)).toBeInTheDocument();
  });

  it("changes the coach default model from the catalog (PUT default + feature + model)", async () => {
    let putBody: unknown = null;
    const keys: Key[] = [{ provider: "anthropic", masked_key: "sk-ant-…4f2a", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true }];
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/models")) return { status: 200, body: MODELS };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: keysBody(keys) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(keys) };
      return { status: 404 };
    });
    await openCoachTab();
    const card = coachCard();

    fireEvent.click(await within(card).findByRole("button", { name: /^change$/i }));
    const list = within(card).getByRole("listbox", { name: /coach model/i });
    // The catalog's own labels, badge, capability tags and dated prices.
    expect(within(list).getByText("Anthropic · models")).toBeInTheDocument();
    expect(within(list).getByText("Prices per MTok, as of Oct 1, 2026")).toBeInTheDocument();
    expect(within(list).getByText("Recommended")).toBeInTheDocument();
    expect(within(list).getByText("$2 in · $10 out")).toBeInTheDocument();
    expect(within(list).getByText("$5 in · $25 out")).toBeInTheDocument();

    fireEvent.click(within(list).getByRole("option", { name: /opus 4\.8/i }));
    await waitFor(() => expect(putBody).toEqual({ provider: "anthropic", default: true, feature: "coach", default_model: "claude-opus-4-8" }));
  });

  it("offers only interview-capable models for the interview default, plus a custom id", async () => {
    let putBody: unknown = null;
    const keys: Key[] = [{ provider: "openai", masked_key: "sk-…9c1e", default_model: "gpt-5.6-luna", name: "GPT-5.6 Luna", enabled: true, is_default: true }];
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/models")) return { status: 200, body: MODELS };
      if (url.includes("/api/coach/key") && init?.method === "PUT") {
        putBody = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: keysBody(keys) };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(keys) };
      return { status: 404 };
    });
    await openCoachTab();
    const card = coachCard();

    fireEvent.click(await within(card).findByRole("button", { name: /^choose$/i }));
    const list = within(card).getByRole("listbox", { name: /interview model/i });
    // GPT-6 Sol carries interview_brain; GPT-5.6 Luna doesn't, so it isn't offered — the
    // server would refuse it (422 model_not_interview_capable).
    expect(within(list).getByRole("option", { name: /gpt-6 sol/i })).toBeInTheDocument();
    expect(within(list).queryByRole("option", { name: /gpt-5\.6 luna/i })).not.toBeInTheDocument();

    // A custom id is accepted (it may be a fine-tune built for exactly this) and shown
    // "cost unknown"; a URL is not a model id.
    fireEvent.click(within(list).getByRole("option", { name: /custom model id/i }));
    const field = within(list).getByLabelText("Model id");
    fireEvent.change(field, { target: { value: "https://proxy.example.com/v1" } });
    expect(field).toHaveAttribute("aria-invalid", "true");
    expect(within(list).getByText("That isn't a model id. Use letters, digits and . _ : - (no URLs).")).toBeInTheDocument();
    expect(within(list).getByRole("button", { name: /^use$/i })).toBeDisabled();

    fireEvent.change(field, { target: { value: "ft:gpt-6-luna:personal:coach" } });
    expect(within(list).getByText("cost unknown")).toBeInTheDocument();
    fireEvent.click(within(list).getByRole("button", { name: /^use$/i }));
    await waitFor(() => expect(putBody).toEqual({ provider: "openai", default: true, feature: "interview", default_model: "ft:gpt-6-luna:personal:coach" }));
  });

  it("shows only the model in use when the catalog can't be read", async () => {
    const keys: Key[] = [{ provider: "anthropic", masked_key: "sk-ant-…4f2a", default_model: "claude-sonnet-5", name: "Sonnet 5", enabled: true, is_default: true }];
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/models")) return { status: 503, body: { error: { code: "unavailable" } } };
      if (url.includes("/api/coach/key")) return { status: 200, body: keysBody(keys) };
      return { status: 404 };
    });
    await openCoachTab();
    const card = coachCard();

    fireEvent.click(await within(card).findByRole("button", { name: /^change$/i }));
    const list = within(card).getByRole("listbox", { name: /coach model/i });
    expect(within(list).getAllByRole("option")).toHaveLength(2); // the model in use + "Custom model id…"
    expect(within(list).getByRole("option", { name: /claude-sonnet-5/ })).toBeInTheDocument();
    // No price and no date are invented from a catalog we never read.
    expect(within(list).queryByText(/Prices per MTok/)).not.toBeInTheDocument();
    expect(within(list).getByText("cost unknown")).toBeInTheDocument();
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

    fireEvent.click(await screen.findByRole("tab", { name: /your ai coach/i }));
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

    fireEvent.click(await screen.findByRole("tab", { name: /reminders/i }));
    fireEvent.click(await screen.findByRole("switch", { name: /daily study reminder/i }));
    fireEvent.click(screen.getByRole("button", { name: /^save$/i }));

    await waitFor(() =>
      expect(patched).toEqual({
        reminders: { daily_reminder_on: false, daily_reminder_time: "20:00", revision_due_alerts_on: true },
      }),
    );
  });

  it("sets a password + offers Connect GitHub in the Sign-in & security card", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/me/password")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { ok: true } };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: { keys: [], connected: false, default_provider: "" } };
      return { status: 404 };
    });
    renderApp("/xlearn/settings");

    fireEvent.click(await screen.findByRole("tab", { name: /sign-in & security/i }));
    await screen.findByRole("heading", { name: /sign-in & security/i });
    // No password yet → "Set password"; GitHub not linked → a top-level Connect (link mode) form.
    const connect = screen.getByRole("button", { name: /^connect$/i });
    expect(connect.closest("form")?.getAttribute("action")).toMatch(/\/api\/v1\/auth\/github\/start\?link=1$/);

    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: "brand-new-pass" } });
    fireEvent.click(screen.getByRole("button", { name: /set password/i }));
    await waitFor(() => expect(posted).toEqual({ new_password: "brand-new-pass" }));
  });

  it("a password change that revokes every session (reauth) sends the learner to sign in again", async () => {
    // identity revokes every session on a password change (m1-04): after the POST, the old
    // cookie is dead — GET /me 401s. The SPA drops the cached account and routes to /auth
    // with a notice (a stale cached /me would bounce the learner straight back in).
    let revoked = false;
    let posted: unknown = null;
    const me = authedMe("dsa");
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me/password") && init?.method === "POST") {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        revoked = true;
        return { status: 200, body: { ok: true, reauth: true } };
      }
      if (url.endsWith("/api/me")) {
        return revoked
          ? { status: 401, body: { error: { code: "unauthenticated" } } }
          : { status: 200, body: { ...me, account: { ...me.account, has_password: true } } };
      }
      if (url.includes("/api/coach/key")) return { status: 200, body: { keys: [], connected: false, default_provider: "" } };
      return { status: 404 };
    });
    renderApp("/xlearn/settings?tab=account");

    await screen.findByRole("heading", { name: /sign-in & security/i });
    fireEvent.change(screen.getByLabelText(/current password/i), { target: { value: "old-password" } });
    fireEvent.change(screen.getByLabelText(/^new password/i), { target: { value: "brand-new-pass" } });
    fireEvent.click(screen.getByRole("button", { name: /^change password$/i }));

    expect(await screen.findByText("Password changed — sign in again.")).toBeInTheDocument();
    expect(posted).toEqual({ current_password: "old-password", new_password: "brand-new-pass" });
    expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /sign-in & security/i })).not.toBeInTheDocument();
  });

  it("asks to retry a password change when identity is busy (429 too_many_requests)", async () => {
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me/password") && init?.method === "POST") return { status: 429, body: { error: { code: "too_many_requests" } } };
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/coach/key")) return { status: 200, body: { keys: [], connected: false, default_provider: "" } };
      return { status: 404 };
    });
    renderApp("/xlearn/settings?tab=account");

    await screen.findByRole("heading", { name: /sign-in & security/i });
    fireEvent.change(screen.getByLabelText(/password/i), { target: { value: "brand-new-pass" } });
    fireEvent.click(screen.getByRole("button", { name: /set password/i }));
    expect(await screen.findByText("Too many attempts right now — try again in a moment.")).toBeInTheDocument();
    // Still on Settings — a throttled attempt is not a sign-out.
    expect(screen.getByRole("heading", { name: /sign-in & security/i })).toBeInTheDocument();
  });

  it("opens the tab named by ?tab= (the coach deep link)", async () => {
    settingsMock();
    renderApp("/xlearn/settings?tab=coach");

    expect(await screen.findByRole("heading", { name: /your ai coach/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /your ai coach/i })).toHaveAttribute("aria-selected", "true");
    expect(screen.queryByRole("heading", { name: /study budget/i })).not.toBeInTheDocument();
  });

  it("opens the Sign-in tab when returning from the GitHub link flow (?linked=)", async () => {
    settingsMock();
    renderApp("/xlearn/settings?linked=github");

    expect(await screen.findByRole("heading", { name: /sign-in & security/i })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: /sign-in & security/i })).toHaveAttribute("aria-selected", "true");
  });
});
