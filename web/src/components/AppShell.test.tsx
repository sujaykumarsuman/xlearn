import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, installFetchMock, restoreFetch } from "../test/fetchMock";

function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe("AppShell + routing (authenticated)", () => {
  beforeEach(() => {
    // Authenticated + onboarded so app routes render the shell (not onboarding).
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 200, body: authedMe("dsa") } : { status: 404 }));
  });
  afterEach(restoreFetch);

  it("renders the shell chrome and the routed screen", async () => {
    renderAt("/xlearn/dsa/dashboard");
    expect(await screen.findByRole("navigation")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /roadmap/i })).toBeInTheDocument();
    expect(await screen.findByRole("heading", { level: 1, name: "Today" })).toBeInTheDocument();
  });

  it("marks the active nav item for the current route", async () => {
    renderAt("/xlearn/dsa/revision");
    await screen.findByRole("navigation");
    const active = document.querySelector(".xl-nav__item--active");
    expect(active?.textContent).toContain("Revision");
  });

  it("renders params-driven screens", async () => {
    renderAt("/xlearn/dsa/problem/16");
    expect(await screen.findByRole("heading", { level: 1, name: "Problem #16" })).toBeInTheDocument();
  });

  it("expanded: the path switcher opens the dropdown", async () => {
    renderAt("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /DSA/i }));
    expect(screen.getByText("Browse all paths")).toBeInTheDocument();
  });

  it("collapsed: the path switcher navigates to Catalog instead of the cramped menu", async () => {
    renderAt("/xlearn/dsa/dashboard");
    await screen.findByRole("navigation");
    const checkbox = document.getElementById("xl-collapse") as HTMLInputElement;
    checkbox.checked = true;
    fireEvent.click(screen.getByRole("button", { name: /DSA/i }));
    expect(await screen.findByRole("heading", { level: 1, name: "Learning paths" })).toBeInTheDocument();
    expect(screen.queryByText("Browse all paths")).not.toBeInTheDocument();
  });

  it("renders all 11 app routes (plus the 404) inside the shell without crashing", async () => {
    const paths = [
      "/xlearn",
      "/xlearn/dsa",
      "/xlearn/dsa/dashboard",
      "/xlearn/dsa/week/2",
      "/xlearn/dsa/concept/sliding-window",
      "/xlearn/dsa/problem/16",
      "/xlearn/dsa/revision",
      "/xlearn/dsa/mistakes",
      "/xlearn/dsa/mock",
      "/xlearn/dsa/progress",
      "/xlearn/settings",
      "/xlearn/nope-404",
    ];
    for (const path of paths) {
      const { unmount } = renderAt(path);
      expect(await screen.findByRole("navigation")).toBeInTheDocument();
      expect(document.querySelector(".xl-app")).not.toBeNull();
      unmount();
    }
  });
});

describe("auth gating", () => {
  afterEach(restoreFetch);

  it("redirects unauthenticated app-route visits to the standalone /auth screen", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderAt("/xlearn/dsa/dashboard");
    // Auth screen (no app shell) with the OAuth buttons.
    expect(await screen.findByRole("button", { name: /continue with github/i })).toBeInTheDocument();
    expect(document.querySelector(".xl-app")).toBeNull();
  });

  it("redirects an authenticated-but-un-onboarded user from an app route into onboarding", async () => {
    // Authenticated (200) but path_chosen=null → must land in onboarding, not the app.
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 200, body: authedMe(null) } : { status: 404 }));
    renderAt("/xlearn/dsa/dashboard");
    expect(await screen.findByRole("heading", { name: /pick your path/i })).toBeInTheDocument();
    expect(document.querySelector(".xl-app")).toBeNull();
  });

  it("keeps the user in the app and shows an error when sign-out fails", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/auth/logout")) return { status: 502, body: { error: { code: "upstream" } } };
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/dashboard");
    await screen.findByRole("navigation");
    fireEvent.click(screen.getByRole("button", { name: /sign out/i }));
    // A failed logout surfaces an error and does NOT navigate to /auth.
    expect(await screen.findByText(/couldn’t sign out/i)).toBeInTheDocument();
    expect(screen.getByRole("navigation")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /continue with github/i })).not.toBeInTheDocument();
  });
});
