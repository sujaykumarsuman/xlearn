import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
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
    expect(form?.getAttribute("action")).toMatch(/\/api\/auth\/github\/start$/);
    expect(form?.getAttribute("method")).toBe("post");
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
    expect(screen.getByText(/DSA Interview Mastery/)).toBeInTheDocument();
    expect(screen.getByText(/Coming soon/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /continue/i }));

    // After persisting, the flow advances to the (inert) step 2 shell.
    expect(await screen.findByRole("heading", { name: /set your study budget/i })).toBeInTheDocument();
    expect(stepPosted).toEqual({ step: "path", path_chosen: "dsa" });
  });

  it("redirects a returning, onboarded user into the app", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 200, body: authedMe("dsa") } : { status: 404 }));
    renderApp("/xlearn/auth");

    // Lands in the app shell rather than showing onboarding.
    expect(await screen.findByRole("navigation")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: /pick your path/i })).not.toBeInTheDocument();
  });
});
