import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, enrolled, installFetchMock, restoreFetch } from "../test/fetchMock";

function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

const PATHS = {
  paths: [
    {
      slug: "dsa",
      title: "Data Structures & Algorithms",
      status: "active",
      summary: "From arrays to graphs and DP.",
      problem_total: 151,
      week_total: 16,
    },
    {
      slug: "system-design",
      title: "System Design Interviews",
      status: "coming_soon",
      summary: "Scalable systems and trade-offs.",
      problem_total: 40,
      week_total: 12,
    },
    {
      slug: "go-concurrency",
      title: "Go Concurrency Patterns",
      status: "coming_soon",
      summary: "Goroutines and channels.",
      problem_total: 60,
      week_total: 8,
    },
  ],
};

describe("Catalog screen", () => {
  afterEach(restoreFetch);

  it("renders a not-started active path (Start CTA) and coming-soon paths from the API", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") }; // no enrollment
      if (url.endsWith("/api/paths")) return { status: 200, body: PATHS };
      return { status: 404 };
    });
    renderApp("/xlearn/");

    // Active path hero: not started yet → "Not started" + a Start CTA + the totals line.
    expect(await screen.findByRole("heading", { name: /Data Structures & Algorithms/ })).toBeInTheDocument();
    expect(screen.getByText("Not started")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /start path/i })).toBeInTheDocument();
    expect(screen.getByText(/16 weeks · 151 problems · Go-first/)).toBeInTheDocument();

    // Coming-soon cards driven by the API (not hard-coded).
    expect(screen.getByText("System Design Interviews")).toBeInTheDocument();
    expect(screen.getByText("Go Concurrency Patterns")).toBeInTheDocument();
    expect(screen.getAllByText("Coming soon").length).toBe(2);
  });

  it("shows the started summary (Day N + Continue) once the path is enrolled", async () => {
    // Started today → deterministically Day 1 regardless of the wall clock.
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa", true, enrolled("dsa", new Date().toISOString())) };
      if (url.endsWith("/api/paths")) return { status: 200, body: PATHS };
      if (url.endsWith("/api/dashboard"))
        return { status: 200, body: { stats: { streak: { current: 3, longest: 5 }, solved: { count: 4, total: 151 }, revisionsDue: 2, mock: { best: 0, last: 0, count: 0, target: 24 } }, plan: [], week: null, revisions: null, weakArea: null, reminders: [] } };
      return { status: 404 };
    });
    renderApp("/xlearn/");

    // Started → the hero shows the current day (badge + summary) + a Continue link.
    expect(await screen.findByRole("heading", { name: /Data Structures & Algorithms/ })).toBeInTheDocument();
    expect(screen.getAllByText(/Day 1/).length).toBeGreaterThan(0);
    // Active streak arrives from the async dashboard agg.
    expect(await screen.findByText(/3-day/)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /continue/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /start path/i })).not.toBeInTheDocument();
  });

  it("shows an error panel with retry when the paths request fails", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths")) return { status: 500, body: { error: { code: "internal" } } };
      return { status: 404 };
    });
    renderApp("/xlearn/");

    expect(await screen.findByText(/Couldn’t load the catalog/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /retry/i })).toBeInTheDocument();
  });

  it("shows an empty state when the API returns no paths", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths")) return { status: 200, body: { paths: [] } };
      return { status: 404 };
    });
    renderApp("/xlearn/");

    expect(await screen.findByText(/No learning paths yet/)).toBeInTheDocument();
    // Not the populated body: no "More paths" header, no engine footer.
    expect(screen.queryByText(/More paths/)).not.toBeInTheDocument();
  });
});
