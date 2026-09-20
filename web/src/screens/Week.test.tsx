import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

const emptyTouches = () => [1, 2, 3, 4, 5].map((level) => ({ level, dueDate: null, result: "none" }));

const WEEK2 = {
  week: { n: 2, title: "Two Pointers & Sliding Window", thesis: "Collapse nested loops into a single sweep." },
  phase: { order: 1, name: "Fundamentals", theme: "Arrays.", week_from: 1, week_to: 3 },
  path: { slug: "dsa", title: "DSA Interview Mastery", problem_total: 151, week_total: 16 },
  concepts: [
    { slug: "sliding-window", title: "Sliding Window" },
    { slug: "two-pointers", title: "Two Pointers" },
  ],
  problems: [
    { id: "12", path_slug: "dsa", week_n: 2, title: "Longest Substring Without Repeating Characters", difficulty: "med", pattern: "Sliding Window", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
    { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "Two Pointers", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
    { id: "18", path_slug: "dsa", week_n: 2, title: "Minimum Window Substring", difficulty: "hard", pattern: "Sliding Window", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
  ],
  userState: {
    week: { solved: 0, coreTotal: 3, byDifficulty: { easy: 0, med: 2, hard: 1 }, populated: false },
    problems: {
      "12": { status: "available", lastOutcome: null, currentTouch: 0, touches: emptyTouches() },
      "16": { status: "available", lastOutcome: null, currentTouch: 0, touches: emptyTouches() },
      "18": { status: "available", lastOutcome: null, currentTouch: 0, touches: emptyTouches() },
    },
  },
};

describe("Week screen", () => {
  afterEach(restoreFetch);

  it("renders the eyebrow, thesis, concepts and the problem list from the aggregation", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths/dsa/weeks/2")) return { status: 200, body: WEEK2 };
      return { status: 404 };
    });
    const { container } = renderApp("/xlearn/dsa/week/2");

    // Eyebrow folds in phase + week_total; the h1 is the week title.
    expect(await screen.findByText("Week 2 of 16 · Phase 1 Fundamentals")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1, name: "Two Pointers & Sliding Window" })).toBeInTheDocument();
    expect(screen.getByText(/Collapse nested loops/)).toBeInTheDocument();

    // Concept cards link to the concept route, carrying the week context.
    const conceptLink = screen
      .getAllByRole("link")
      .find((l) => l.getAttribute("href") === "/xlearn/dsa/concept/sliding-window?week=2");
    expect(conceptLink).toBeTruthy();

    // Problem rows render with difficulty tokens + the honest "Available" status.
    expect(screen.getByText("3Sum")).toBeInTheDocument();
    expect(screen.getByText("Minimum Window Substring")).toBeInTheDocument();
    // Difficulty tokens (scoped by class — the rail's "Med"/"Hard" labels reuse the words).
    expect(container.querySelectorAll(".xl-diff--med").length).toBe(2);
    expect(container.querySelectorAll(".xl-diff--hard").length).toBe(1);
    expect(screen.getAllByText("Available").length).toBe(3);

    // Progress meter is the placeholder rollup: 0 solved of 3 core (never faked).
    expect(screen.getByText("/ 3 core solved")).toBeInTheDocument();
    // A problem row routes to the S05 problem stub.
    expect(screen.getByRole("link", { name: /3Sum/ })).toHaveAttribute("href", "/xlearn/dsa/problem/16");
  });

  it("filters to reinforcement and shows an honest empty state (no reinforcement seeded)", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths/dsa/weeks/2")) return { status: 200, body: WEEK2 };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/week/2");

    await screen.findByText("3Sum");
    await userEvent.click(screen.getByRole("button", { name: "Reinforcement" }));

    expect(screen.getByText("No reinforcement problems this week.")).toBeInTheDocument();
    expect(screen.queryByText("3Sum")).not.toBeInTheDocument();
  });

  it("shows an explicit not-found state for a malformed week (no query, no blank page)", async () => {
    const calls: string[] = [];
    installFetchMock((url) => {
      calls.push(url);
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/week/abc");

    expect(await screen.findByText(/isn’t a valid week/)).toBeInTheDocument();
    // The aggregation query must not fire for a malformed week number.
    expect(calls.some((u) => u.includes("/api/paths/dsa/weeks/"))).toBe(false);
    // No broken "Week NaN of 16" header slips through.
    expect(screen.queryByText(/NaN/)).not.toBeInTheDocument();
  });

  it("shows a not-found message for an unknown week", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths/dsa/weeks/99"))
        return { status: 404, body: { error: { code: "not_found" } } };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/week/99");

    expect(await screen.findByText(/isn’t part of this path yet/)).toBeInTheDocument();
  });
});
