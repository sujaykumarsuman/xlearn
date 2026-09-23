import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, installFetchMock, restoreFetch } from "../test/fetchMock";

/** A mid-onboarding /me: path chosen, and budget_set/completed as given. */
function onboardingMe(budgetSet: boolean, completed: boolean) {
  return {
    account: authedMe("dsa").account,
    onboarding: { path_chosen: "dsa", budget_set: budgetSet, key_added: false, completed },
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

  it("shows onboarding step 1 and persists the chosen path on Continue", async () => {
    let stepPosted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe(null) };
      if (url.endsWith("/api/onboarding/step")) {
        stepPosted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: false, key_added: false, completed: false } } };
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
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, key_added: false, completed: false } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /set your study budget/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /continue/i }));

    await waitFor(() =>
      expect(posted).toEqual({ step: "budget", study_budget: { weekday_minutes: 90, weekend_band: "3-4" } }),
    );
    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();
  });

  it("step 2 seeds from an already-saved budget instead of overwriting it with defaults", async () => {
    let posted: unknown = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) {
        // Budget already saved (150 / "5") but not yet marked → resumes at step 2 and
        // must reflect the saved value, so Continue re-posts it (not the 90/"3-4" default).
        const me = {
          account: { ...authedMe("dsa").account, study_budget: { weekday_minutes: 150, weekend_band: "5" } },
          onboarding: { path_chosen: "dsa", budget_set: false, key_added: false, completed: false },
        };
        return { status: 200, body: me };
      }
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, key_added: false, completed: false } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /set your study budget/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /continue/i }));
    await waitFor(() => expect(posted).toEqual({ step: "budget", study_budget: { weekday_minutes: 150, weekend_band: "5" } }));
  });

  it("resumes at step 3 and completes onboarding on Finish", async () => {
    let posted: unknown = null;
    let finished = false;
    installFetchMock((url, init) => {
      // path + budget done, not completed → the flow resumes at step 3.
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(true, finished) };
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        finished = true;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, key_added: false, completed: true } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    expect(await screen.findByRole("heading", { name: /power up your coach/i })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /finish & enter xlearn/i }));

    await waitFor(() => expect(posted).toEqual({ step: "finish" }));
  });

  it("Skip for now also completes onboarding without a key", async () => {
    let posted: unknown = null;
    let finished = false;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: onboardingMe(true, finished) };
      if (url.endsWith("/api/onboarding/step")) {
        posted = init?.body ? JSON.parse(String(init.body)) : null;
        finished = true;
        return { status: 200, body: { onboarding: { path_chosen: "dsa", budget_set: true, key_added: false, completed: true } } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/auth");

    fireEvent.click(await screen.findByRole("button", { name: /skip for now/i }));
    await waitFor(() => expect(posted).toEqual({ step: "finish" }));
  });
});
