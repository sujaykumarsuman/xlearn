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

const SLIDING_WINDOW = {
  concept: {
    slug: "sliding-window",
    path_slug: "dsa",
    title: "Sliding Window",
    when_to_use_md: "Reach for a sliding window for longest or shortest contiguous ranges.",
    body_md:
      "The window keeps a range and a bit of state.\n\n## The idea\n\nExpand **right**, shrink while invalid.\n\n> [!warning] The width is `right - left + 1`, not `right - left`.",
    code_template: "func longest(s string) int {\n    return 0\n}",
  },
};

const WEEK2 = {
  week: { n: 2, title: "Two Pointers & Sliding Window", thesis: "t2" },
  phase: { order: 1, name: "Fundamentals", theme: "Arrays.", week_from: 1, week_to: 3 },
  path: { slug: "dsa", title: "DSA", problem_total: 151, week_total: 16 },
  concepts: [],
  problems: [
    { id: "12", path_slug: "dsa", week_n: 2, title: "Longest Substring Without Repeating Characters", difficulty: "med", pattern: "Sliding Window", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
    { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "Two Pointers", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
    { id: "18", path_slug: "dsa", week_n: 2, title: "Minimum Window Substring", difficulty: "hard", pattern: "Sliding Window", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
  ],
  userState: { week: { solved: 0, coreTotal: 3, byDifficulty: { easy: 0, med: 2, hard: 1 }, populated: false }, problems: {} },
};

describe("Concept screen", () => {
  afterEach(restoreFetch);

  it("renders the reading, when-to-use callout, code template and pattern practice links", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/concepts/sliding-window")) return { status: 200, body: SLIDING_WINDOW };
      if (url.endsWith("/api/paths/dsa/weeks/2")) return { status: 200, body: WEEK2 };
      return { status: 404 };
    });
    const { container } = renderApp("/xlearn/dsa/concept/sliding-window?week=2");

    // Title + week-aware eyebrow.
    expect(await screen.findByRole("heading", { level: 1, name: "Sliding Window" })).toBeInTheDocument();
    expect(screen.getByText("Pattern · Week 2")).toBeInTheDocument();

    // when_to_use_md → the teal callout.
    expect(screen.getByText(/Reach for a sliding window/)).toBeInTheDocument();

    // body_md → sanitized markdown: a heading + a "Watch out" callout from the admonition.
    expect(screen.getByRole("heading", { name: "The idea" })).toBeInTheDocument();
    expect(screen.getByText("Watch out")).toBeInTheDocument();

    // The Go code template renders in the xl-code block; the C++ tab is stubbed/disabled.
    const code = container.querySelector("pre.xl-code");
    expect(code?.textContent).toContain("func longest");
    expect(screen.getByRole("button", { name: "C++" })).toBeDisabled();

    // Right rail: real problems whose pattern matches this concept, and a Practice
    // now button into the S05 problem stub — no fabricated links.
    const links = screen.getAllByRole("link");
    const hrefs = links.map((l) => l.getAttribute("href"));
    expect(hrefs).toContain("/xlearn/dsa/problem/12");
    expect(hrefs).toContain("/xlearn/dsa/problem/18");
    expect(hrefs).not.toContain("/xlearn/dsa/problem/16"); // Two Pointers, not this pattern
    expect(screen.getByRole("link", { name: /Practice now/ })).toHaveAttribute("href", "/xlearn/dsa/problem/12");
  });

  it("without a week context, still renders the reading and points practice at the roadmap", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/concepts/sliding-window")) return { status: 200, body: SLIDING_WINDOW };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/concept/sliding-window");

    expect(await screen.findByRole("heading", { level: 1, name: "Sliding Window" })).toBeInTheDocument();
    expect(screen.getByText("Pattern")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Practice now/ })).toHaveAttribute("href", "/xlearn/dsa");
  });
});
