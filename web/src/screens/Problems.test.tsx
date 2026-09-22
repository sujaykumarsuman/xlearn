import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { authedMe, enrolled, installFetchMock, restoreFetch } from "../test/fetchMock";

function renderApp(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path], basename: "/xlearn" });
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

const PROBLEMS = {
  problems: [
    { id: "1", path_slug: "dsa", week_n: 1, title: "Contains Duplicate", difficulty: "easy", pattern: "Hashing", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
    { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "Two Pointers", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
  ],
};

describe("Problems arena", () => {
  afterEach(restoreFetch);

  it("lists every problem grouped by week, and marks weeks ahead of the frontier", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa", true, enrolled("dsa")) };
      if (url.endsWith("/api/paths/dsa/problems")) return { status: 200, body: PROBLEMS };
      // Frontier = week 1, so week 2 is "ahead · free practice".
      if (url.endsWith("/api/progress"))
        return { status: 200, body: { summary: { solved: 0, total: 151, streak: { current: 0, longest: 0 } }, phases: [], patterns: [], enrolled: true, currentWeek: 1, revisionsDue: 0 } };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/problems");

    expect(await screen.findByRole("heading", { level: 1, name: "Problems" })).toBeInTheDocument();
    // Both problems are listed (await the async problem index) and link to their workspace.
    expect(await screen.findByText("Contains Duplicate")).toBeInTheDocument();
    // Arena links open the workspace in practice mode (?practice=1) — open, doesn't count.
    expect(screen.getByRole("link", { name: /3Sum/ })).toHaveAttribute("href", "/xlearn/dsa/problem/16?practice=1");
    // Week 1 is current; week 2 is ahead → free practice label.
    expect(screen.getByText("Current")).toBeInTheDocument();
    expect(screen.getByText(/Ahead · free practice/)).toBeInTheDocument();
  });
});
