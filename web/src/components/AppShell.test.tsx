import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
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

  it("shows live Practice-loop badge counts (reviews due · open mistakes)", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/revision/due")) return { status: 200, body: { items: [], dueCount: 2 } };
      if (url.endsWith("/api/mistakes"))
        return { status: 200, body: { mistakes: [], openCount: 5, closedCount: 0, closeThreshold: 0, categories: [] } };
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/dashboard");

    const nav = await screen.findByRole("navigation");
    await waitFor(() => {
      expect(within(nav).getByRole("link", { name: /revision/i })).toHaveTextContent("2");
      expect(within(nav).getByRole("link", { name: /mistakes/i })).toHaveTextContent("5");
    });
  });

  it("routes the sidebar brand home", async () => {
    renderAt("/xlearn/dsa/dashboard");
    const brand = await screen.findByRole("link", { name: /xlearn — home/i });
    expect(brand).toHaveAttribute("href", "/xlearn");
  });

  it("renders params-driven screens", async () => {
    // The Problem workspace fetches its BFF aggregate; give it a minimal one.
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/problems/16")) {
        return {
          status: 200,
          body: {
            problem: { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "Two Pointers", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
            sections: [{ stage: "attempt", kind: "summary", order: 1, body_md: "Find all triplets that sum to zero.", code: "" }],
            state: { problemId: "16", status: "available", stageReached: "", unlockedStages: ["attempt"], currentTouch: 0, lastOutcome: null, firstSolvedAt: null, revealedEarly: false, timer: null },
          },
        };
      }
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/problem/16");
    expect(await screen.findByRole("heading", { level: 1, name: "3Sum" })).toBeInTheDocument();
  });

  it("opens the top-bar curriculum switcher dropdown (F001)", async () => {
    renderAt("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /switch curriculum/i }));
    expect(screen.getByText("Browse all paths")).toBeInTheDocument();
  });

  it("renders every app route without crashing — curriculum routes get the sidebar, hub routes don't (F001)", async () => {
    const curriculum = [
      "/xlearn/dsa",
      "/xlearn/dsa/dashboard",
      "/xlearn/dsa/week/2",
      "/xlearn/dsa/concept/sliding-window",
      "/xlearn/dsa/problem/16",
      "/xlearn/dsa/revision",
      "/xlearn/dsa/mistakes",
      "/xlearn/dsa/mock",
      "/xlearn/dsa/progress",
    ];
    // A multi-segment unknown still hits the in-shell NotFound. (A single-segment unknown
    // like /xlearn/nope is now the PUBLIC profile route, tested in UserDashboard.test.)
    const hub = ["/xlearn", "/xlearn/settings", "/xlearn/nope/404"];

    for (const path of curriculum) {
      const { unmount } = renderAt(path);
      expect(await screen.findByRole("navigation")).toBeInTheDocument();
      expect(document.querySelector(".xl-app")).not.toBeNull();
      unmount();
    }
    for (const path of hub) {
      const { unmount } = renderAt(path);
      // The sidebar-less shell: brand lockup present, no sidebar nav.
      expect(await screen.findByRole("link", { name: /all paths/i })).toBeInTheDocument();
      expect(document.querySelector(".xl-app")).not.toBeNull();
      expect(document.querySelector(".xl-side")).toBeNull();
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
    // Open the account menu (button-disclosure) before reaching Sign out.
    fireEvent.click(screen.getByRole("button", { name: /account menu/i }));
    fireEvent.click(screen.getByRole("button", { name: /sign out/i }));
    // A failed logout surfaces an error and does NOT navigate to /auth.
    expect(await screen.findByText(/couldn’t sign out/i)).toBeInTheDocument();
    expect(screen.getByRole("navigation")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /continue with github/i })).not.toBeInTheDocument();
  });
});
