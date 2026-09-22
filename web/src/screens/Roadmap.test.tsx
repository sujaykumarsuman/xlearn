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

const DSA = {
  path: {
    slug: "dsa",
    title: "Data Structures & Algorithms",
    status: "active",
    summary: "From arrays to graphs and DP.",
    problem_total: 151,
    week_total: 16,
  },
  phases: [
    { order: 1, name: "Fundamentals", theme: "Arrays, hashing, and the two-pointer mind.", week_from: 1, week_to: 3 },
    { order: 2, name: "Core Data Structures", theme: "The structures interviewers reach for.", week_from: 4, week_to: 8 },
    { order: 3, name: "Advanced Patterns", theme: "Where medium becomes hard.", week_from: 9, week_to: 12 },
    { order: 4, name: "Interview Mastery", theme: "Perform under real interview conditions.", week_from: 13, week_to: 16 },
  ],
  weeks: [
    { n: 1, title: "Arrays, Strings, Hashing", thesis: "t1", easy: 3, med: 3, hard: 0, total: 6 },
    { n: 2, title: "Two Pointers & Sliding Window", thesis: "t2", easy: 0, med: 2, hard: 1, total: 3 },
    { n: 3, title: "Stacks & Monotonic Stack", thesis: "t3", easy: 0, med: 0, hard: 1, total: 1 },
    { n: 4, title: "Linked Lists & Design", thesis: "t4", easy: 0, med: 1, hard: 0, total: 1 },
    { n: 5, title: "Trees & BST", thesis: "Recursion over structure.", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 6, title: "Heaps & Intervals", thesis: "t6", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 7, title: "Tries & Backtracking", thesis: "t7", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 8, title: "Graphs & Topological Sort", thesis: "t8", easy: 0, med: 1, hard: 0, total: 1 },
    { n: 9, title: "Shortest Paths", thesis: "t9", easy: 0, med: 1, hard: 0, total: 1 },
    { n: 10, title: "Dynamic Programming I", thesis: "t10", easy: 0, med: 1, hard: 0, total: 1 },
    { n: 11, title: "Dynamic Programming II", thesis: "t11", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 12, title: "Greedy & Bit Manipulation", thesis: "t12", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 13, title: "Mixed sets + first mocks", thesis: "t13", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 14, title: "Timed company sets", thesis: "t14", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 15, title: "Mock intensives", thesis: "t15", easy: 0, med: 0, hard: 0, total: 0 },
    { n: 16, title: "Final rounds & retro", thesis: "t16", easy: 0, med: 0, hard: 0, total: 0 },
  ],
};

describe("Roadmap screen", () => {
  afterEach(restoreFetch);

  it("renders the 4 phases and 16 weeks from the API, with difficulty tokens", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths/dsa")) return { status: 200, body: DSA };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa");

    // All four phase names render.
    expect(await screen.findByRole("heading", { name: "Fundamentals" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Core Data Structures" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Advanced Patterns" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Interview Mastery" })).toBeInTheDocument();

    // First and last week cards render.
    expect(screen.getByText(/Week 1 · Arrays, Strings, Hashing/)).toBeInTheDocument();
    expect(screen.getByText(/Week 16 · Final rounds & retro/)).toBeInTheDocument();

    // Difficulty tokens: the seeded difficulty counts render (Easy/Med/Hard).
    expect(screen.getByText("3 Easy")).toBeInTheDocument(); // week 1
    expect(screen.getByText("2 Med")).toBeInTheDocument(); // week 2
    expect(screen.getAllByText("1 Hard").length).toBeGreaterThanOrEqual(1); // weeks 2 & 3

    // Overall progress is a placeholder until practice state exists.
    expect(screen.getByText("0 / 151 solved")).toBeInTheDocument();
  });
});
