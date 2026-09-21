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

const past = new Date(Date.now() - 24 * 3600_000).toISOString();
const future = new Date(Date.now() + 6 * 24 * 3600_000).toISOString();

function item(over: Partial<Record<string, unknown>>) {
  return {
    itemId: "it-x",
    problemId: "3",
    touchLevel: 1,
    dayLabel: "Day 1",
    dueDate: past,
    due: true,
    mockMode: false,
    status: "pending",
    problem: { id: "3", title: "Two Sum", difficulty: "easy", pattern: "Complement lookup", week_n: 1 },
    ...over,
  };
}

const QUEUE = {
  items: [
    item({ itemId: "it-1", problemId: "3", touchLevel: 1, dayLabel: "Day 1" }),
    item({
      itemId: "it-2",
      problemId: "16",
      touchLevel: 3,
      dayLabel: "Day 7",
      problem: { id: "16", title: "3Sum", difficulty: "med", pattern: "Two pointers", week_n: 2 },
    }),
    item({
      itemId: "it-3",
      problemId: "2",
      touchLevel: 4,
      dayLabel: "Day 21",
      dueDate: future,
      due: false,
      mockMode: true,
      problem: { id: "2", title: "Valid Anagram", difficulty: "easy", pattern: "HashMap", week_n: 1 },
    }),
  ],
  dueCount: 2,
};

describe("Revision queue", () => {
  afterEach(restoreFetch);

  it("renders the prioritised due queue grouped by touch day, with an upcoming tail", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/revision/due")) return { status: 200, body: QUEUE };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/revision");

    expect(await screen.findByRole("heading", { level: 1, name: "Revision queue" })).toBeInTheDocument();
    // "N due today" from dueCount (await the query resolving).
    expect(await screen.findByText(/2 due today/)).toBeInTheDocument();
    // Grouped by touch day.
    expect(screen.getByRole("heading", { name: /Due · Day 1/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Due · Day 7/ })).toBeInTheDocument();
    // Enriched problem titles.
    expect(screen.getByText("Two Sum")).toBeInTheDocument();
    expect(screen.getByText("3Sum")).toBeInTheDocument();
    // The not-due item is in "Coming up", not the due groups.
    expect(screen.getByRole("heading", { name: "Coming up" })).toBeInTheDocument();
    expect(screen.getByText("Valid Anagram")).toBeInTheDocument();
    expect(screen.getByText("Not due yet")).toBeInTheDocument();
  });

  it("re-solving submits the auto-score inputs and shows the pass → advance result", async () => {
    const fetchMock = installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/revision/it-1/score") && init?.method === "POST") {
        return {
          status: 200,
          body: {
            itemId: "it-1",
            problemId: "3",
            touchLevel: 1,
            autoPass: true,
            status: "passed",
            mockMode: false,
            reset: false,
            nextTouchLevel: 2,
            nextDayLabel: "Day 3",
            nextDueDate: future,
          },
        };
      }
      if (url.includes("/api/revision/due")) return { status: 200, body: QUEUE };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/revision");

    await screen.findByRole("button", { name: /Start next review/ });
    await userEvent.click(screen.getByRole("button", { name: /Start next review/ }));

    // The re-solve panel appears on a 20:00 timer.
    expect(await screen.findByText("20:00")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: /Submit re-solve for auto-score/ }));

    // The auto-score result panel: pass advances to Day 3.
    expect(await screen.findByText(/Passed — advances to Day 3/)).toBeInTheDocument();
    // The score endpoint was actually called with a POST.
    expect(
      fetchMock.mock.calls.some(([u, i]) => String(u).includes("/api/v1/revision/it-1/score") && (i as RequestInit)?.method === "POST"),
    ).toBe(true);
  });

  it("a pass advancing into Day 21 flags mock conditions (from the next touch, not the scored one)", async () => {
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/revision/it-1/score") && init?.method === "POST") {
        // Scored a Day-7 touch (not mock); it advances to Day 21 (mock). mockMode is
        // the scored touch's flag (false) — the panel must use the NEXT level instead.
        return {
          status: 200,
          body: { itemId: "it-1", problemId: "3", touchLevel: 3, autoPass: true, status: "passed", mockMode: false, reset: false, nextTouchLevel: 4, nextDayLabel: "Day 21", nextDueDate: future },
        };
      }
      if (url.includes("/api/revision/due")) return { status: 200, body: QUEUE };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/revision");

    await screen.findByRole("button", { name: /Start next review/ });
    await userEvent.click(screen.getByRole("button", { name: /Start next review/ }));
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    expect(await screen.findByText(/Next review Day 21 \(mock conditions\)/)).toBeInTheDocument();
  });

  it("submits a failing pattern time when the learner never affirms 'Named the pattern'", async () => {
    let scoreBody: Record<string, unknown> | null = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/revision/it-1/score") && init?.method === "POST") {
        scoreBody = JSON.parse(String(init?.body ?? "{}"));
        return { status: 200, body: { itemId: "it-1", problemId: "3", touchLevel: 1, autoPass: false, status: "failed", mockMode: false, reset: true, nextTouchLevel: 1, nextDayLabel: "Day 1", nextDueDate: future } };
      }
      if (url.includes("/api/revision/due")) return { status: 200, body: QUEUE };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/revision");

    await screen.findByRole("button", { name: /Start next review/ });
    await userEvent.click(screen.getByRole("button", { name: /Start next review/ }));
    // Submit WITHOUT tapping "Named the pattern": the payload must carry a value that
    // fails the < 120s check, not the small elapsed time.
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    expect(scoreBody).not.toBeNull();
    expect((scoreBody as unknown as { namedPatternSecs: number }).namedPatternSecs).toBe(120);
  });

  it("a miss resets the problem to Day 1", async () => {
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/revision/it-1/score") && init?.method === "POST") {
        return {
          status: 200,
          body: {
            itemId: "it-1",
            problemId: "3",
            touchLevel: 1,
            autoPass: false,
            status: "failed",
            mockMode: false,
            reset: true,
            nextTouchLevel: 1,
            nextDayLabel: "Day 1",
            nextDueDate: future,
          },
        };
      }
      if (url.includes("/api/revision/due")) return { status: 200, body: QUEUE };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/revision");

    await screen.findByRole("button", { name: /Start next review/ });
    await userEvent.click(screen.getByRole("button", { name: /Start next review/ }));
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    expect(await screen.findByText(/reset to Day 1/)).toBeInTheDocument();
  });

  it("shows the empty state when nothing is due", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/revision/due")) return { status: 200, body: { items: [], dueCount: 0 } };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/revision");

    expect(await screen.findByText(/Queue clear/)).toBeInTheDocument();
  });
});
