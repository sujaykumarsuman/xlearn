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

const adaProfile = {
  user: { username: "ada", displayName: "Ada Lovelace", joinedAt: "2026-01-01T00:00:00Z", region: "UTC+05:30" },
  totals: { solved: 12, streak: { current: 5, longest: 9 } },
  mock: { count: 2, best: 24, average: 22 },
  heatmap: { days: [{ date: "2026-09-20", solves: 1, reviews: 2 }] },
  courses: [
    {
      slug: "dsa",
      title: "Data Structures & Algorithms",
      solved: 12,
      total: 151,
      pct: 8,
      phases: [{ order: 1, name: "Fundamentals", theme: "t", weekFrom: 1, weekTo: 3, solved: 3, total: 10, clean: 2, rough: 1, assisted: 0, miss: 0 }],
      patterns: [{ name: "Hashing", solved: 2, total: 4, pct: 50 }],
    },
  ],
};

describe("UserDashboard (public /xlearn/<username>)", () => {
  afterEach(restoreFetch);

  it("renders a public profile with no login and no PII", async () => {
    installFetchMock((url) => {
      // Anonymous viewer: /me is a 401 and must NOT gate this public route.
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/u/ada")) return { status: 200, body: adaProfile };
      return { status: 404 };
    });
    renderApp("/xlearn/ada");

    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    expect(screen.getByText("@ada")).toBeInTheDocument();
    expect(screen.getByText("Data Structures & Algorithms")).toBeInTheDocument();
    // Account-wide solved total + the coarse region (F009 review).
    expect(screen.getByText("12")).toBeInTheDocument();
    expect(screen.getByText("UTC+05:30")).toBeInTheDocument();

    // Public page: NO app shell, and an anonymous viewer gets a Sign-in affordance.
    expect(document.querySelector(".xl-app")).toBeNull();
    expect(screen.getByRole("link", { name: /sign in/i })).toBeInTheDocument();

    // No PII rendered anywhere.
    expect(screen.queryByText(/ada@example\.com/)).not.toBeInTheDocument();
  });

  it("shows the authenticated header (account menu, no Sign in) when the viewer is signed in", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/u/ada")) return { status: 200, body: adaProfile };
      if (url.includes("/api/coach/key")) return { status: 200, body: { keys: [] } };
      return { status: 404 };
    });
    renderApp("/xlearn/ada");

    expect(await screen.findByText("Ada Lovelace")).toBeInTheDocument();
    // Authenticated viewer → the app's account menu, and NO "Sign in" link.
    expect(await screen.findByRole("button", { name: /account menu/i })).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: /^sign in$/i })).not.toBeInTheDocument();
  });

  it("collapses and expands per-course detail", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/u/ada")) return { status: 200, body: adaProfile };
      return { status: 404 };
    });
    renderApp("/xlearn/ada");

    // The first course is expanded by default → its detail is visible.
    expect(await screen.findByText("Completion by phase")).toBeInTheDocument();
    // Collapsing the row hides the detail; the overview (title) stays.
    fireEvent.click(screen.getByRole("button", { name: /Data Structures & Algorithms/i }));
    expect(screen.queryByText("Completion by phase")).not.toBeInTheDocument();
    expect(screen.getByText("Data Structures & Algorithms")).toBeInTheDocument();
  });

  it("shows a friendly 404 for an unclaimed username", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 401, body: { error: { code: "unauthenticated" } } };
      if (url.includes("/api/u/ghost")) return { status: 404, body: { error: { code: "not_found" } } };
      return { status: 404 };
    });
    renderApp("/xlearn/ghost");

    expect(await screen.findByText(/no profile for @ghost/i)).toBeInTheDocument();
    expect(document.querySelector(".xl-app")).toBeNull();
  });
});
