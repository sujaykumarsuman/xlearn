import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
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

const PATHS = {
  paths: [
    {
      slug: "dsa",
      title: "DSA Interview Mastery",
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

  it("renders the active DSA path and the coming-soon paths from the API", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths")) return { status: 200, body: PATHS };
      return { status: 404 };
    });
    renderApp("/xlearn/");

    // Active path hero.
    expect(await screen.findByRole("heading", { name: /DSA Interview Mastery/ })).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText(/16 weeks · 151 problems · Go-first/)).toBeInTheDocument();

    // Coming-soon cards driven by the API (not hard-coded).
    expect(screen.getByText("System Design Interviews")).toBeInTheDocument();
    expect(screen.getByText("Go Concurrency Patterns")).toBeInTheDocument();
    expect(screen.getAllByText("Coming soon").length).toBe(2);
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
});
