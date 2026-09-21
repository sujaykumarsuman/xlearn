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
  reminders: [{ id: "r1", kind: "revision_due", dueAt: past }],
  weakArea: {
    weekOf: "2026-09-21", topCategory: "off_by_one", topCount: 3, counts: { off_by_one: 3 }, entries: [],
  },
};

describe("Dashboard (Today)", () => {
  afterEach(restoreFetch);

  it("renders the revisions-due panel and the weak-area card", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/dashboard")) return { status: 200, body: DASHBOARD };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/dashboard");

    expect(await screen.findByRole("heading", { level: 1, name: "Today" })).toBeInTheDocument();
    // Await a data-dependent element (the h1 renders before the query resolves).
    expect(await screen.findByText("Two Sum")).toBeInTheDocument();
    // Revisions-due panel with the enriched due item.
    expect(screen.getByRole("heading", { name: /Revisions due today/ })).toBeInTheDocument();
    expect(screen.getByText("Day 1")).toBeInTheDocument();
    // Weak-area card.
    expect(screen.getByText("Weak area this week")).toBeInTheDocument();
    expect(screen.getByText("Off-by-one / boundary")).toBeInTheDocument();
    // In-app reminders surfaced (the panel footer + the reminders card).
    expect(screen.getAllByText(/in-app reminder/).length).toBeGreaterThan(0);
  });

  it("shows the empty state when no reviews are due", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/dashboard")) {
        return { status: 200, body: { revisions: { items: [], dueCount: 0 }, reminders: [], weakArea: null } };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/dashboard");

    expect(await screen.findByText(/Queue clear/)).toBeInTheDocument();
    // With no weak area yet, the card prompts to classify.
    expect(screen.getByText(/No weak area yet/)).toBeInTheDocument();
  });
});
