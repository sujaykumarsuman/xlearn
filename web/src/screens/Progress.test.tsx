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

const PROGRESS = {
  summary: {
    solved: 28,
    total: 151,
    streak: { current: 12, longest: 18 },
    retention: { pct: 80, resets: 1, ladders: 5 },
    mock: { count: 3, average: 22, best: 24, last: 24, delta: 2 },
    outcomeMix: { total: 28, clean: 15, rough: 7, assisted: 4, miss: 2 },
  },
  heatmap: { days: [{ date: new Date().toISOString().slice(0, 10), solves: 1, reviews: 2 }] },
  trend: {
    points: [
      { mockId: "m1", setId: "s", problemId: "16", difficulty: "med", date: "2026-09-01", startedAt: "2026-09-01T10:00:00Z", total35: 20 },
      { mockId: "m2", setId: "s", problemId: "17", difficulty: "med", date: "2026-09-08", startedAt: "2026-09-08T10:00:00Z", total35: 24 },
    ],
    targets: { w13: 24, w15: 28, pre: 30 },
  },
  phases: [
    { order: 1, name: "Fundamentals", theme: "t", weekFrom: 1, weekTo: 3, solved: 28, total: 36 },
    { order: 2, name: "Core Data Structures", theme: "t", weekFrom: 4, weekTo: 8, solved: 0, total: 45 },
  ],
  patterns: [
    { name: "Hashing / frequency", solved: 6, total: 7, pct: 86 },
    { name: "Sliding window", solved: 1, total: 4, pct: 28 },
  ],
  weakArea: null,
};

describe("Progress", () => {
  afterEach(restoreFetch);

  it("renders the tiles, phase completion, pattern mastery and outcome mix", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/progress")) return { status: 200, body: PROGRESS };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/progress");

    expect(await screen.findByRole("heading", { level: 1, name: "Your progress" })).toBeInTheDocument();
    // Stat tiles.
    expect(await screen.findByText("Problems solved")).toBeInTheDocument();
    expect(screen.getByText("Day-7 retention")).toBeInTheDocument();
    expect(screen.getByText("80%")).toBeInTheDocument();
    // Completion by phase.
    expect(screen.getByText("Fundamentals")).toBeInTheDocument();
    expect(screen.getByText("28 / 36")).toBeInTheDocument();
    // Pattern mastery.
    expect(screen.getByText("Hashing / frequency")).toBeInTheDocument();
    expect(screen.getByText("Sliding window")).toBeInTheDocument();
    // Outcome mix legend.
    expect(screen.getByText("Clean")).toBeInTheDocument();
    expect(screen.getByText("28 first-solves")).toBeInTheDocument();
  });

  it("shows the empty mock-trend state with no scored mocks", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/progress")) {
        return {
          status: 200,
          body: {
            ...PROGRESS,
            summary: { ...PROGRESS.summary, mock: { count: 0, average: 0, best: 0, last: 0, delta: 0 } },
            trend: { points: [], targets: { w13: 24, w15: 28, pre: 30 } },
          },
        };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/progress");

    expect(await screen.findByText(/Take your first mock/)).toBeInTheDocument();
    expect(screen.getByText("No mocks yet")).toBeInTheDocument();
  });
});
