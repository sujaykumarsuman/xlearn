import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { DSA_PATH, ZZ_FIXTURE_PATH, ZZ_SOON_PATH, catalog } from "../test/courses";
import { authedMe, installFetchMock, restoreFetch } from "../test/fetchMock";

/** A mid-onboarding /me: path chosen, and budget_set/completed as given. */
function onboardingMe(budgetSet: boolean, completed: boolean) {
  return {
    account: authedMe("dsa").account,
    onboarding: { path_chosen: "dsa", budget_set: budgetSet, completed },
  };
}

function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe("Auth screen", () => {
  afterEach(restoreFetch);

  it("shows OAuth sign-in when unauthenticated, with correct form actions", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderApp("/xlearn/auth");

    const github = await screen.findByRole("button", { name: /continue with github/i });
    expect(github).toBeInTheDocument();
    // Google is deferred in v1 — no Google button should render.
    expect(screen.queryByRole("button", { name: /continue with google/i })).not.toBeInTheDocument();

    const form = github.closest("form");
    expect(form).not.toBeNull();
    // BASE_URL is "/" under vitest and "/xlearn/" in the build, so match the
    // suffix (the API-relative path) rather than the absolute prefix.
    expect(form?.getAttribute("action")).toMatch(/\/api\/v1\/auth\/github\/start$/);
    expect(form?.getAttribute("method")).toBe("post");
  });

  it("signs up with email + password (POST /auth/signup)", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/auth/signup")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { ok: true } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    await screen.findByRole("button", { name: /continue with github/i });
    fireEvent.click(screen.getByRole("button", { name: /^sign up$/i }));
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "new@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "hunter2hunter" } });
    fireEvent.click(screen.getByRole("button", { name: /create account/i }));

    await waitFor(() => expect(posted).toEqual({ email: "new@example.com", password: "hunter2hunter" }));
  });

  it("signs in with email + password (POST /auth/login)", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/auth/login")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { ok: true } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    await screen.findByRole("button", { name: /continue with github/i });
    // Sign in is the default mode; the identifier field accepts email or username (F009).
    fireEvent.change(screen.getByLabelText("Email or username"), { target: { value: "me@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "opensesame" } });
    fireEvent.click(screen.getByRole("button", { name: /^log in$/i }));

    await waitFor(() => expect(posted).toEqual({ email: "me@example.com", password: "opensesame" }));
  });

  it("surfaces an error when the email is already registered", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/auth/signup")) return { status: 409, body: { error: { code: "email_taken" } } };
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    await screen.findByRole("button", { name: /continue with github/i });
    fireEvent.click(screen.getByRole("button", { name: /^sign up$/i }));
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "taken@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "hunter2hunter" } });
    fireEvent.click(screen.getByRole("button", { name: /create account/i }));

    expect(await screen.findByText(/already registered/i)).toBeInTheDocument();
  });

  it("says sign-up is invite-only when signup is closed (email + GitHub)", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/auth/signup")) return { status: 403, body: { error: { code: "signup_closed" } } };
      return { status: 404 };
    });
    renderApp("/xlearn/auth?error=signup_closed");

    expect(await screen.findByText(/this github account isn’t connected/i)).toHaveTextContent(/invite-only right now/i);
    fireEvent.click(screen.getByRole("button", { name: /^sign up$/i }));
    fireEvent.change(screen.getByLabelText("Email"), { target: { value: "new@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "hunter2hunter" } });
    fireEvent.click(screen.getByRole("button", { name: /create account/i }));

    expect(await screen.findByText(/^xLearn is invite-only right now\.$/i)).toBeInTheDocument();
  });

  it("explains a refused GitHub sign-in onto a password account", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderApp("/xlearn/auth?error=account_exists_password");

    const alert = await screen.findByText(/already uses a password/i);
    expect(alert).toHaveTextContent(/sign in with your password, then connect github from settings/i);
  });

  it("explains a suspended account on the OAuth callback (?error=account_unavailable)", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderApp("/xlearn/auth?error=account_unavailable");

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("This account isn’t available. Contact the owner if you think this is a mistake.");
  });

  it("shows the password-changed notice (?notice=password_changed) as a status, not an error", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderApp("/xlearn/auth?notice=password_changed");

    expect(await screen.findByText("Password changed — sign in again.")).toHaveAttribute("role", "status");
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    // The sign-in form is right there to use.
    expect(screen.getByRole("button", { name: /continue with github/i })).toBeInTheDocument();
  });

  it("ignores an unknown ?notice= code", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderApp("/xlearn/auth?notice=bogus");

    await screen.findByRole("button", { name: /continue with github/i });
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(screen.queryByText(/password changed/i)).not.toBeInTheDocument();
  });

  it("asks the learner to retry when identity is busy (429 too_many_requests)", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/auth/login")) return { status: 429, body: { error: { code: "too_many_requests" } } };
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    await screen.findByRole("button", { name: /continue with github/i });
    fireEvent.change(screen.getByLabelText("Email or username"), { target: { value: "me@example.com" } });
    fireEvent.change(screen.getByLabelText("Password"), { target: { value: "opensesame" } });
    fireEvent.click(screen.getByRole("button", { name: /^log in$/i }));

    expect(await screen.findByText("Too many attempts right now — try again in a moment.")).toBeInTheDocument();
  });

  it("shows onboarding step 1 and persists the chosen path on Continue", async () => {
    let stepPosted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe(null) };
      if (url.endsWith("/api/onboarding/step")) {
        stepPosted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: false, completed: false } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /pick your path/i })).toBeInTheDocument();
    expect(screen.getByText(/Data Structures & Algorithms/)).toBeInTheDocument();
    expect(screen.getByText(/Coming soon/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /continue/i }));

    // After persisting, the flow advances to the (inert) step 2 shell.
    expect(await screen.findByRole("heading", { name: /set your study budget/i })).toBeInTheDocument();
    expect(stepPosted).toEqual({ step: "path", path_chosen: "dsa" });
  });

  it("renders step 1's DSA card exactly as v1 from the catalog", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 200, body: authedMe(null) } : { status: 404 }));
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /pick your path/i })).toBeInTheDocument();
    expect(screen.getByText("Start with DSA — more paths are on the way.")).toBeInTheDocument();
    const card = screen.getByRole("button", { name: /Data Structures & Algorithms/ });
    expect(card).toHaveAttribute("aria-pressed", "true");
    expect(card).toHaveClass("ds-card", "ds-card--teal");
    expect(within(card).getByText("16 weeks · 151 problems · Go-first")).toBeInTheDocument();
    expect(screen.getByText("System Design · Go Concurrency · +3")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /continue/i })).toBeEnabled();
  });

  it("step 1 lists every active course and persists the one picked", async () => {
    let stepPosted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe(null) };
      if (url.endsWith("/api/paths")) return { status: 200, body: catalog(DSA_PATH, ZZ_FIXTURE_PATH, ZZ_SOON_PATH) };
      if (url.endsWith("/api/onboarding/step")) {
        stepPosted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { onboarding: { path_chosen: "zz-fixture", budget_set: false, completed: false } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    const dsa = await screen.findByRole("button", { name: /Data Structures & Algorithms/ });
    const zz = screen.getByRole("button", { name: /Fixture Course/ });
    // The first active course is the default; the coming-soon one isn't a card.
    expect(dsa).toHaveAttribute("aria-pressed", "true");
    expect(zz).toHaveAttribute("aria-pressed", "false");
    expect(screen.queryByRole("button", { name: /Soon Course/ })).not.toBeInTheDocument();
    expect(screen.getByText("Pick one to start — more paths are on the way.")).toBeInTheDocument();

    fireEvent.click(zz);
    expect(zz).toHaveAttribute("aria-pressed", "true");
    fireEvent.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() => expect(stepPosted).toEqual({ step: "path", path_chosen: "zz-fixture" }));
  });

  it("redirects a returning, onboarded user into the app", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths"))
        return {
          status: 200,
          body: { paths: [{ slug: "dsa", title: "Data Structures & Algorithms", status: "active", summary: "s", problem_total: 151, week_total: 16 }] },
        };
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    // Lands on the sidebar-less Catalog home (F001) rather than showing onboarding.
    expect(await screen.findByRole("heading", { name: /learning paths/i })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /pick your path/i })).not.toBeInTheDocument();
  });

  it("resumes at step 2 and persists the study budget, then advances to step 3", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      // path chosen, budget not set → the flow resumes at step 2.
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(false, false) };
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, completed: false } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /set your study budget/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /continue/i }));

    await waitFor(() =>
      expect(posted).toEqual({ step: "budget", study_budget: { weekday_minutes: 90, weekend_band: "3-4" } }),
    );
    // Step 3 is now the username step (F009 review), ahead of the coach step.
    expect(await screen.findByRole("heading", { name: /claim your username/i })).toBeInTheDocument();
  });

  it("step 2 seeds from an already-saved budget instead of overwriting it with defaults", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) {
        // Budget already saved (150 / "5") but not yet marked → resumes at step 2 and
        // must reflect the saved value, so Continue re-posts it (not the 90/"3-4" default).
        const me = {
          account: { ...authedMe("dsa").account, study_budget: { weekday_minutes: 150, weekend_band: "5" } },
          onboarding: { path_chosen: "dsa", budget_set: false, completed: false },
        };
        return { status: 200, body: me };
      }
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, completed: false } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /set your study budget/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() => expect(posted).toEqual({ step: "budget", study_budget: { weekday_minutes: 150, weekend_band: "5" } }));
  });

  it("claims a username in the onboarding username step, then advances to the coach step", async () => {
    let usernamePosted: unknown = null;
    installFetchMock((url, init) => {
      // path + budget done, not completed → the flow resumes at the username step (3).
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(true, false) };
      if (url.includes("/api/username/available")) return { status: 200, body: { available: true } };
      if (url.includes("/api/me/username")) {
        usernamePosted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { ok: true, username: "ada-l" } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /claim your username/i })).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "ada-l" } });
    const claim = await screen.findByRole("button", { name: /claim & continue/i });
    await waitFor(() => expect(claim).toBeEnabled());
    fireEvent.click(claim);

    await waitFor(() => expect(usernamePosted).toEqual({ username: "ada-l" }));
    // Advances to the (optional) coach step.
    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();
  });

  it("resumes at step 3, skips username, and completes onboarding on Finish", async () => {
    let posted: unknown = null;
    let finished = false;
    installFetchMock((url, init) => {
      // path + budget done, not completed → the flow resumes at the username step (3).
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(true, finished) };
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        finished = true;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, completed: true } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    // Skip the username step → coach step → Finish.
    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i }));
    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /finish & enter xlearn/i }));

    await waitFor(() => expect(posted).toEqual({ step: "finish" }));
  });

  it("Skip for now on both optional steps still completes onboarding", async () => {
    let posted: unknown = null;
    let finished = false;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(true, finished) };
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        finished = true;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, completed: true } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i })); // username step
    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /skip for now/i })); // coach step
    await waitFor(() => expect(posted).toEqual({ step: "finish" }));
  });

  /** coachStepMock resumes at the username step (path + budget done) and records the
   *  coach-key PUT + onboarding/step POST bodies in order. `putStatus` fails the PUT. */
  function coachStepMock(putStatus = 200) {
    const calls: Array<{ kind: "put" | "finish"; body: unknown }> = [];
    let finished = false;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(true, finished) };
      if (url.endsWith("/api/coach/key") && init?.method === "PUT") {
        calls.push({ kind: "put", body: JSON.parse(String(init.body)) });
        if (putStatus !== 200) return { status: putStatus, body: { error: { code: "invalid_key", message: "key is too long" } } };
        return { status: 200, body: { keys: [], connected: true, default_provider: "anthropic" } };
      }
      if (url.endsWith("/api/onboarding/step")) {
        calls.push({ kind: "finish", body: JSON.parse(String(init?.body)) });
        finished = true;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, completed: true } } };
      }
      return { status: 404 };
    });
    return calls;
  }

  it("coach step offers only the supported providers (no Google)", async () => {
    coachStepMock();
    renderApp("/xlearn/auth");
    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i })); // username step
    const group = await screen.findByRole("group", { name: /coach provider/i });
    expect(within(group).getAllByRole("button").map((b) => b.textContent)).toEqual(["Anthropic", "OpenAI"]);
  });

  it("saves the typed key via PUT /coach/key before completing onboarding", async () => {
    const calls = coachStepMock();
    renderApp("/xlearn/auth");
    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i })); // username step
    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "OpenAI" }));
    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "  sk-openai-test-1234  " } });
    fireEvent.click(screen.getByRole("button", { name: /finish & enter xlearn/i }));

    await waitFor(() => expect(calls.map((c) => c.kind)).toEqual(["put", "finish"]));
    // Same body shape Settings sends: provider + key + the provider's first model + its label.
    expect(calls[0]!.body).toEqual({ provider: "openai", key: "sk-openai-test-1234", default_model: "gpt-5.6-sol", name: "GPT-5.6 Sol" });
    expect(calls[1]!.body).toEqual({ step: "finish" });
  });

  it("a failed key save shows an error and does not finish; Skip still completes without the key", async () => {
    const calls = coachStepMock(422);
    renderApp("/xlearn/auth");
    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i })); // username step
    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("API key"), { target: { value: "sk-ant-bad" } });
    fireEvent.click(screen.getByRole("button", { name: /finish & enter xlearn/i }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(/couldn’t save your key \(key is too long\)/i);
    expect(alert).toHaveTextContent(/skip for now/i);
    // Onboarding is NOT marked finished and the learner stays on the coach step.
    expect(calls.map((c) => c.kind)).toEqual(["put"]);
    expect(screen.getByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();

    // Skip completes onboarding without re-sending the key.
    fireEvent.click(screen.getByRole("button", { name: /skip for now/i }));
    await waitFor(() => expect(calls.map((c) => c.kind)).toEqual(["put", "finish"]));
  });

  it("Finish with no key typed completes onboarding without calling PUT /coach/key", async () => {
    const calls = coachStepMock();
    renderApp("/xlearn/auth");
    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i })); // username step
    fireEvent.click(await screen.findByRole("button", { name: /finish & enter xlearn/i }));
    await waitFor(() => expect(calls.map((c) => c.kind)).toEqual(["finish"]));
  });
});
