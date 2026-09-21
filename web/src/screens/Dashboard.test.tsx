import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
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

const past = new Date(Date.now() - 24 * 3600_000).toISOString();

const DASHBOARD = {
  stats: {
    streak: { current: 12, longest: 18 },
    solved: { count: 28, total: 151 },
    revisionsDue: 1,
    mock: { best: 24, last: 24, count: 3, target: 24 },
  },
  plan: [
    { kind: "review", itemId: "it-1", problemId: "3", title: "Two Sum", dayLabel: "Day 1", touchLevel: 1, mockMode: false },
    { kind: "problem", problemId: "16", title: "3Sum", difficulty: "med", pattern: "Two pointers", status: "available" },
  ],
  week: {
    n: 2,
    title: "Two Pointers",
    solved: 2,
    total: 5,
    problems: [
      { problemId: "16", title: "3Sum", status: "available" },
      { problemId: "17", title: "Container", status: "solved" },
    ],
  },
  revisions: {
    items: [
      {
        itemId: "it-1", problemId: "3", touchLevel: 1, dayLabel: "Day 1", dueDate: past,
        due: true, mockMode: false, status: "pending",
        problem: { id: "3", title: "Two Sum", difficulty: "easy", pattern: "Complement lookup", week_n: 1 },
      },
    ],
    dueCount: 1,
  },
  weakArea: { weekOf: "2026-09-21", topCategory: "off_by_one", topCount: 3, counts: { off_by_one: 3 }, entries: [] },
  reminders: [{ id: "r1", kind: "revision_due", dueAt: past }],
};

describe("Dashboard (Today)", () => {
  afterEach(restoreFetch);

  it("renders quick stats, the plan with reviews first, and the revisions panel", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/dashboard")) return { status: 200, body: DASHBOARD };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/dashboard");

    expect(await screen.findByRole("heading", { level: 1, name: "Today" })).toBeInTheDocument();
    // Quick stats.
    expect(await screen.findByText("28")).toBeInTheDocument(); // solved
    expect(screen.getByText("/ 151")).toBeInTheDocument();
    // The plan leads with the review (reviews before new work), then the new problem.
    expect(screen.getByText(/re-solve/)).toBeInTheDocument();
    expect(screen.getByText(/New problem/)).toBeInTheDocument();
    expect(screen.getByText("Medium")).toBeInTheDocument();
    // Week-progress panel.
    expect(screen.getByText(/Week 2 progress/)).toBeInTheDocument();
    expect(screen.getByText(/2 of 5 core problems solved/)).toBeInTheDocument();
    // Revisions-due panel + weak-area card.
    expect(screen.getByRole("heading", { name: /Revisions due today/ })).toBeInTheDocument();
    expect(screen.getByText("Off-by-one / boundary")).toBeInTheDocument();
  });

  it("shows the caught-up state when there is no plan and no reviews", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/dashboard")) {
        return {
          status: 200,
          body: {
            stats: { streak: { current: 0, longest: 0 }, solved: { count: 0, total: 151 }, revisionsDue: 0, mock: { best: 0, last: 0, count: 0, target: 24 } },
            plan: [],
            week: null,
            revisions: { items: [], dueCount: 0 },
            weakArea: null,
            reminders: [],
          },
        };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/dashboard");

    expect(await screen.findByText(/All caught up/)).toBeInTheDocument();
    expect(screen.getByText(/no reviews due/)).toBeInTheDocument();
    expect(screen.getByText(/No weak area yet/)).toBeInTheDocument();
  });
});
