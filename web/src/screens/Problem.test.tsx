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

const PROBLEM = { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "Two Pointers", leetcode_url: "https://leetcode.com/problems/3sum/", neetcode_url: "", is_reinforcement: false };
const STATEMENT = { stage: "attempt", kind: "summary", order: 1, body_md: "Find all unique triplets that sum to zero.", code: "" };
const HINT = { stage: "hint", kind: "key_observation", order: 1, body_md: "Sort, then two pointers.", code: "" };
const SOLUTION = { stage: "solution", kind: "code", order: 1, body_md: "", code: "func threeSum(){}" };

function agg(status: string, stageReached: string, unlocked: string[], sections: unknown[], extra: Record<string, unknown> = {}) {
  return {
    problem: PROBLEM,
    sections,
    state: {
      problemId: "16",
      status,
      stageReached,
      unlockedStages: unlocked,
      currentTouch: 0,
      lastOutcome: null,
      firstSolvedAt: null,
      revealedEarly: false,
      timer: null,
      ...extra,
    },
  };
}

describe("Problem workspace", () => {
  afterEach(restoreFetch);

  it("shows the statement + Start attempt, and withholds locked hint/solution", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/problems/16")) return { status: 200, body: agg("available", "", ["attempt"], [STATEMENT]) };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/problem/16");

    expect(await screen.findByRole("heading", { level: 1, name: "3Sum" })).toBeInTheDocument();
    expect(screen.getByText("Find all unique triplets that sum to zero.")).toBeInTheDocument();
    // Medium difficulty token (amber).
    expect(document.querySelector(".xl-diff--med")).not.toBeNull();
    // The gate: locked content is absent (the gateway never delivered it).
    expect(screen.queryByText("Sort, then two pointers.")).not.toBeInTheDocument();
    expect(screen.queryByText("Hints")).not.toBeInTheDocument();
    expect(screen.queryByText("Solution")).not.toBeInTheDocument();
    // The entry CTA.
    expect(screen.getByRole("button", { name: /Start attempt/ })).toBeInTheDocument();
  });

  it("starting an attempt calls practice and reveals the timed attempt state", async () => {
    let started = false;
    const fetchMock = installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/problems/16/attempt/start") && init?.method === "POST") {
        started = true;
        return { status: 200, body: { state: agg("attempting", "attempt", ["attempt"], [STATEMENT]).state } };
      }
      if (url.includes("/api/problems/16")) {
        const timer = { kind: "attempt", deadlineAt: new Date(Date.now() + 900_000).toISOString(), remainingSeconds: 900, expired: false };
        return { status: 200, body: started ? agg("attempting", "attempt", ["attempt"], [STATEMENT], { timer }) : agg("available", "", ["attempt"], [STATEMENT]) };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/problem/16");

    await screen.findByRole("button", { name: /Start attempt/ });
    await userEvent.click(screen.getByRole("button", { name: /Start attempt/ }));

    // The reveal control appears once attempting; the outcome picker too.
    expect(await screen.findByRole("button", { name: /show a hint/ })).toBeInTheDocument();
    expect(screen.getByText("Log your outcome")).toBeInTheDocument();
    // Practice was actually called to start the attempt.
    expect(fetchMock.mock.calls.some(([u, i]) => String(u).includes("/attempt/start") && (i as RequestInit)?.method === "POST")).toBe(true);
  });

  it("after the solution is revealed early, offers re-implement and flags the penalty", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/problems/16")) {
        return { status: 200, body: agg("attempting", "solution", ["attempt", "hint", "solution"], [STATEMENT, HINT, SOLUTION], { revealedEarly: true }) };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/problem/16");

    // Hint + solution content are now delivered.
    expect(await screen.findByText("Sort, then two pointers.")).toBeInTheDocument();
    expect(screen.getByText("func threeSum(){}")).toBeInTheDocument();
    // Re-implement CTA + the early-reveal note in the outcome panel.
    expect(screen.getByRole("button", { name: /Re-implement from memory/ })).toBeInTheDocument();
    expect(screen.getByText(/revealed the solution early/)).toBeInTheDocument();
  });

  it("a solved problem shows the confirmation with the five-touch schedule + owed re-attempt", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/problems/16")) {
        return {
          status: 200,
          body: agg("solved", "solution", ["attempt", "hint", "solution"], [STATEMENT, HINT, SOLUTION], {
            lastOutcome: "miss",
            firstSolvedAt: "2026-09-21T00:00:00Z",
            revealedEarly: true,
          }),
        };
      }
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/problem/16");

    expect(await screen.findByText("Logged as Miss")).toBeInTheDocument();
    expect(screen.getByText(/Five-touch revision schedule created/)).toBeInTheDocument();
    expect(screen.getByText(/re-attempt queued for Day 3/)).toBeInTheDocument();
  });
});
