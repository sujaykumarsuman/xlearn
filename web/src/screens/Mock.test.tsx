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

const PHASE_META = [
  { index: 0, label: "Clarify", range: "0–5", startMin: 0, endMin: 5, grow: 5, prompt: "Restate the problem." },
  { index: 1, label: "Brute force", range: "5–10", startMin: 5, endMin: 10, grow: 5, prompt: "Describe the naive solution." },
  { index: 2, label: "Observation → plan", range: "10–18", startMin: 10, endMin: 18, grow: 8, prompt: "Find the key insight." },
  { index: 3, label: "Code", range: "18–33", startMin: 18, endMin: 33, grow: 15, prompt: "Implement steadily." },
  { index: 4, label: "Trace + edges", range: "33–40", startMin: 33, endMin: 40, grow: 7, prompt: "Dry-run the edges." },
  { index: 5, label: "Complexity + follow-ups", range: "40–45", startMin: 40, endMin: 45, grow: 5, prompt: "State the complexity." },
];

function phases(currentIdx: number) {
  return PHASE_META.map((p) => ({
    ...p,
    state: p.index < currentIdx ? "past" : p.index === currentIdx ? "current" : "upcoming",
  }));
}

const targets = { w13: 24, w15: 28, pre: 30 };

function liveSession() {
  const now = Date.now();
  return {
    id: "mk-1",
    status: "live",
    setId: "set-07",
    problemId: "16",
    difficulty: "med",
    date: "2026-09-21",
    startedAt: new Date(now).toISOString(),
    deadlineAt: new Date(now + 45 * 60 * 1000).toISOString(),
    total35: null,
    notes: "",
    rail: { elapsedSeconds: 0, remainingSeconds: 2700, overtime: false, phaseIndex: 0, phaseLabel: "Clarify", phases: phases(0) },
    dimensions: [],
    targets,
    problem: { id: "16", title: "3Sum", difficulty: "med", pattern: "Two Pointers", week_n: 2 },
  };
}

const DIMS = [
  { key: "communication", name: "Communication" },
  { key: "problem_understanding", name: "Problem understanding" },
  { key: "brute_force", name: "Brute force" },
  { key: "optimisation", name: "Optimisation" },
  { key: "code_quality", name: "Code quality" },
  { key: "edge_cases", name: "Edge cases" },
  { key: "complexity", name: "Complexity" },
];

function scoredSession(total: number) {
  const s = liveSession();
  return {
    ...s,
    status: "scored",
    total35: total,
    notes: "named the pattern late",
    dimensions: DIMS.map((d, i) => ({ ...d, score: (i % 5) + 1 })),
  };
}

const TREND = {
  points: [
    { mockId: "mk-0", setId: "set-05", problemId: "16", difficulty: "med", date: "2026-09-07", startedAt: "2026-09-07T10:00:00Z", total35: 20 },
    { mockId: "mk-1", setId: "set-07", problemId: "16", difficulty: "med", date: "2026-09-21", startedAt: "2026-09-21T10:00:00Z", total35: 24 },
  ],
  targets,
};

describe("Mock interview", () => {
  afterEach(restoreFetch);

  it("renders the setup with the rail reference and rubric summary", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/mock");

    expect(await screen.findByRole("heading", { level: 1, name: "Mock interview" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Start 45-minute mock/ })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Medium" })).toHaveAttribute("aria-pressed", "true");
    // The 45-minute rail reference + 7-dimension card.
    expect(screen.getByText("Key observation → plan")).toBeInTheDocument();
    expect(screen.getByText(/SCORED ON 7 DIMENSIONS/)).toBeInTheDocument();
  });

  it("starts a 45-minute mock and shows the live phase rail + server timer", async () => {
    const fetchMock = installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/mocks") && init?.method === "POST") return { status: 201, body: liveSession() };
      if (url.includes("/api/mocks/mk-1")) return { status: 200, body: liveSession() };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/mock");

    await userEvent.click(await screen.findByRole("button", { name: /Start 45-minute mock/ }));

    // Live view: the 45:00 clock reference, the problem header, and the current phase.
    expect(await screen.findByText("/ 45:00")).toBeInTheDocument();
    expect(screen.getByText(/3Sum · Two Pointers/)).toBeInTheDocument();
    expect(screen.getByText(/Now · Clarify/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Finish & score/ })).toBeInTheDocument();
    // POST /mocks actually fired.
    expect(fetchMock.mock.calls.some(([u, i]) => String(u).endsWith("/api/v1/mocks") && (i as RequestInit)?.method === "POST")).toBe(true);
  });

  it("finishes, scores the seven dimensions, and shows the /35 result + trend", async () => {
    let scoreBody: Record<string, unknown> | null = null;
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/mocks") && init?.method === "POST") return { status: 201, body: liveSession() };
      if (url.includes("/api/mocks/trend")) return { status: 200, body: TREND };
      if (url.includes("/api/mocks/mk-1/score") && init?.method === "POST") {
        scoreBody = JSON.parse(String(init?.body ?? "{}"));
        return { status: 200, body: scoredSession(24) };
      }
      if (url.includes("/api/mocks/mk-1")) return { status: 200, body: liveSession() };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/mock");

    await userEvent.click(await screen.findByRole("button", { name: /Start 45-minute mock/ }));
    await userEvent.click(await screen.findByRole("button", { name: /Finish & score/ }));

    // Scoring form: all seven dimensions + the submit.
    expect(await screen.findByRole("heading", { level: 1, name: "Rubric" })).toBeInTheDocument();
    expect(screen.getByText("Communication")).toBeInTheDocument();
    expect(screen.getByText("Edge cases")).toBeInTheDocument();
    const submit = screen.getByRole("button", { name: /Submit rubric/ });
    await userEvent.click(submit);

    // Results: the /35 headline + the New mock reset.
    expect(await screen.findByRole("heading", { level: 1, name: "Rubric & score" })).toBeInTheDocument();
    expect(screen.getByText("24")).toBeInTheDocument();
    expect(screen.getAllByText("/ 35").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: /New mock/ })).toBeInTheDocument();

    // The submitted rubric carried exactly seven dimensions.
    expect(scoreBody).not.toBeNull();
    const scores = (scoreBody as unknown as { scores: Record<string, number> }).scores;
    expect(Object.keys(scores)).toHaveLength(7);
  });

  it("renders results directly when the fetched session is already scored", async () => {
    // Simulate a resume: navigating with a pre-scored session id is not a route param
    // here, so drive it through start -> the GET returning a scored session on poll.
    installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/mocks") && init?.method === "POST") return { status: 201, body: scoredSession(30) };
      if (url.includes("/api/mocks/trend")) return { status: 200, body: TREND };
      if (url.includes("/api/mocks/mk-1")) return { status: 200, body: scoredSession(30) };
      return { status: 404 };
    });
    renderApp("/xlearn/dsa/mock");

    await userEvent.click(await screen.findByRole("button", { name: /Start 45-minute mock/ }));
    // A session that comes back scored jumps straight to results.
    expect(await screen.findByRole("heading", { level: 1, name: "Rubric & score" })).toBeInTheDocument();
    expect(screen.getByText("30")).toBeInTheDocument();
  });
});
