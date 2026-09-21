import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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

const future = new Date(Date.now() + 2 * 24 * 3600_000).toISOString();

const MISTAKES = {
  mistakes: [
    {
      id: "m1", problemId: "18", pattern: "Sliding window",
      mistake: "Computed window width as right − left", rootCause: "Off-by-one", insight: "width = right − left + 1",
      category: "off_by_one", status: "open", revisitCount: 1, revisitDate: future, createdAt: "2026-09-20T00:00:00Z",
      problem: { id: "18", title: "Minimum Window Substring", difficulty: "hard", pattern: "Sliding window", week_n: 3 },
    },
    {
      id: "m2", problemId: "04", pattern: "Canonical key",
      mistake: "", rootCause: "", insight: "",
      category: "", status: "open", revisitCount: 0, revisitDate: null, createdAt: "2026-09-20T00:00:00Z",
      problem: { id: "04", title: "Group Anagrams", difficulty: "med", pattern: "Canonical key", week_n: 2 },
    },
    {
      id: "m3", problemId: "05", pattern: "Bucket sort",
      mistake: "Spent 22 min on a heap", rootCause: "Over-engineered", insight: "count → bucket is O(n)",
      category: "time_management", status: "closed", revisitCount: 2, revisitDate: null, createdAt: "2026-09-19T00:00:00Z",
      problem: { id: "05", title: "Top K Frequent", difficulty: "med", pattern: "Bucket sort", week_n: 2 },
    },
  ],
  openCount: 2,
  closedCount: 1,
  closeThreshold: 2,
  categories: ["off_by_one", "time_management"],
};

const WEAK_AREA = {
  weekOf: "2026-09-21",
  topCategory: "off_by_one",
  topCount: 1,
  counts: { off_by_one: 1 },
  entries: [MISTAKES.mistakes[0]],
};

function mockAll() {
  return installFetchMock((url) => {
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.includes("/api/mistakes")) return { status: 200, body: MISTAKES };
    if (url.includes("/api/weak-area")) return { status: 200, body: WEAK_AREA };
    return { status: 404 };
  });
}

describe("Mistake journal", () => {
  afterEach(restoreFetch);

  it("renders the journal, header counts and the weak-area banner", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");

    expect(await screen.findByRole("heading", { level: 1, name: "Mistake journal" })).toBeInTheDocument();
    // Enriched problem titles across the full journal.
    expect(await screen.findByText("Minimum Window Substring")).toBeInTheDocument();
    expect(screen.getByText("Group Anagrams")).toBeInTheDocument();
    expect(screen.getByText("Top K Frequent")).toBeInTheDocument();
    // The weekly weak-area banner (its copy is banner-specific).
    expect(screen.getByText("This week's weak area")).toBeInTheDocument();
    expect(screen.getByText(/most-missed area this week/)).toBeInTheDocument();
    // Status cell shows the n/2 progress for an open entry.
    expect(screen.getByText(/1\/2/)).toBeInTheDocument();
  });

  it("shows a category picker for an uncategorised entry", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");
    // The uncategorised entry (#04) renders the 8-category <select> picker.
    const picker = await screen.findByRole("combobox", { name: /Categorise problem 04/ });
    expect(picker).toBeInTheDocument();
    expect(within(picker).getByRole("option", { name: "Off-by-one / boundary" })).toBeInTheDocument();
  });

  it("filters the table by the Open/Closed segmented control", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");
    await screen.findByText("Minimum Window Substring");

    await userEvent.click(screen.getByRole("button", { name: "Closed" }));
    // Only the closed entry remains.
    expect(screen.getByText("Top K Frequent")).toBeInTheDocument();
    expect(screen.queryByText("Minimum Window Substring")).not.toBeInTheDocument();
  });

  it("filters by category chip", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");
    await screen.findByText("Minimum Window Substring");

    // The chip label includes its count; match by the leading text.
    await userEvent.click(screen.getByRole("button", { name: /Off-by-one \/ boundary/ }));
    expect(screen.getByText("Minimum Window Substring")).toBeInTheDocument();
    expect(screen.queryByText("Top K Frequent")).not.toBeInTheDocument();
  });

  it("'Show these N' filters to the weak-area category's open entries (count matches)", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");
    const show = await screen.findByRole("button", { name: /Show these 1/ });
    await userEvent.click(show);
    // Only the off_by_one OPEN entry remains; other-category + closed entries are gone.
    expect(screen.getByText("Minimum Window Substring")).toBeInTheDocument();
    expect(screen.queryByText("Group Anagrams")).not.toBeInTheDocument();
    expect(screen.queryByText("Top K Frequent")).not.toBeInTheDocument();
  });

  it("filter controls expose aria-pressed for the active option", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");
    await screen.findByText("Minimum Window Substring");
    // The segmented status control marks the active option pressed.
    expect(screen.getByRole("button", { name: "All", pressed: true })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open", pressed: false })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Open" }));
    expect(screen.getByRole("button", { name: "Open", pressed: true })).toBeInTheDocument();
  });

  it("re-open note is shown", async () => {
    mockAll();
    renderApp("/xlearn/dsa/mistakes");
    expect(await screen.findByText(/re-opens automatically and re-enters the revision queue at Day 1/)).toBeInTheDocument();
  });
});
