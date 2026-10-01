import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it, vi } from "vitest";
import { routes } from "../router";
import { DSA_PATH, ZZ_FIXTURE_PATH, ZZ_SOON_PATH, catalog } from "../test/courses";
import { authedMe, installFetchMock, jsonResponse, restoreFetch, sseResponse, type RouteHandler } from "../test/fetchMock";

/** The FAB's accessible name (AB01 naming reference: never shortened to "AI" alone). */
const FAB_NAME = "Open Your AI coach (your key)";

function renderApp(initialPath: string) {
  const router = createMemoryRouter(routes, { initialEntries: [initialPath], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

const enabledKey = { keys: [{ provider: "openai", masked_key: "sk-...1234", default_model: "gpt-4o-mini", enabled: true }], connected: true };
const noKey = { keys: [], connected: false };

/** coachMock wires /me + a coach-key state + a thread + a chat handler. */
function coachMock(opts: {
  key?: { keys: unknown[]; connected: boolean };
  thread?: unknown[];
  chat?: (init?: RequestInit) => Response | { status: number; body?: unknown };
}): void {
  const handler: RouteHandler = (url, init) => {
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.endsWith("/api/coach/key")) return { status: 200, body: opts.key ?? noKey };
    if (url.includes("/api/coach/thread")) return { status: 200, body: { messages: opts.thread ?? [] } };
    if (url.endsWith("/api/coach/chat")) return opts.chat ? opts.chat(init) : { status: 200, body: {} };
    return { status: 404 };
  };
  installFetchMock(handler);
}

describe("Coach panel", () => {
  afterEach(restoreFetch);

  it("shows the FAB and opens the panel", async () => {
    coachMock({ key: enabledKey });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    expect(await screen.findByRole("complementary", { name: "Your AI coach (your key)" })).toBeInTheDocument();
  });

  it("shows the empty state and routes to Settings when there is no key", async () => {
    coachMock({ key: noKey });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    expect(await screen.findByText(/your ai coach is off/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /open settings/i }));
    expect(await screen.findByRole("heading", { level: 1, name: "Settings" })).toBeInTheDocument();
  });

  it("streams a reply into the transcript and posts the page context", async () => {
    let chatBody: { context?: string; message?: string } = {};
    coachMock({
      key: enabledKey,
      chat: (init) => {
        chatBody = init?.body ? JSON.parse(String(init.body)) : {};
        return sseResponse([`data: {"delta":"Hi "}\n\n`, `data: {"delta":"there"}\n\n`, `data: {"done":true}\n\n`]);
      },
    });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));

    const input = await screen.findByLabelText(/message coach/i);
    fireEvent.change(input, { target: { value: "how am I doing?" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));

    expect(await screen.findByText("Hi there")).toBeInTheDocument();
    expect(screen.getByText("how am I doing?")).toBeInTheDocument();
    // Course-scoped contexts carry their course (m1-03): the thread key is per course.
    await waitFor(() => expect(chatBody.context).toBe("dsa:dashboard"));
    expect(chatBody.message).toBe("how am I doing?");
  });

  it("loads the existing thread history on open", async () => {
    coachMock({ key: enabledKey, thread: [{ role: "assistant", content: "We spoke about two pointers earlier." }] });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    expect(await screen.findByText(/two pointers earlier/i)).toBeInTheDocument();
  });

  it("routes to Settings when the provider rejects the key mid-stream", async () => {
    coachMock({
      key: enabledKey,
      chat: () => sseResponse([`data: {"error":"provider_auth","message":"rejected"}\n\n`]),
    });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    const input = await screen.findByLabelText(/message coach/i);
    fireEvent.change(input, { target: { value: "hello" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    expect(await screen.findByRole("heading", { level: 1, name: "Settings" })).toBeInTheDocument();
  });

  /** sendAndFail sends a turn that the provider refuses with the given `reason`, and
   *  returns once the panel has rendered its line for it. */
  async function sendAndFail(reason: string, opts: { status?: number; key?: { keys: unknown[]; connected: boolean } } = {}) {
    coachMock({
      key: opts.key ?? enabledKey,
      chat: () => ({
        status: opts.status ?? 429,
        body: { error: { code: "provider_limited", message: "your provider account is out of credit or limited — top up and retry", reason } },
      }),
    });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    fireEvent.change(await screen.findByLabelText(/message coach/i), { target: { value: "hint please" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    return screen.findByRole("alert");
  }

  it.each([
    // Quota and a rate limit share one line: both mean "the account, not the key".
    ["quota", "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry."],
    ["rate_limit", "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry."],
    ["region", "Your provider doesn't serve this region."],
  ])("reason=%s shows its own line and keeps the coach on", async (reason, line) => {
    const alert = await sendAndFail(reason);
    expect(alert).toHaveTextContent(line);
    // No bounce to Settings: the key is fine, so the panel stays usable.
    expect(screen.queryByRole("heading", { level: 1, name: "Settings" })).not.toBeInTheDocument();
    expect(screen.getByLabelText(/message coach/i)).toBeInTheDocument();
  });

  it("reason=model_access offers the catalog switcher instead of blaming the key", async () => {
    const alert = await sendAndFail("model_access");
    expect(alert).toHaveTextContent("This key can't use gpt-4o-mini. Pick another model.");
    // The list opens with the affordance, and the failing model stays listed and marked.
    const list = await screen.findByRole("listbox", { name: /coach model/i });
    expect(within(list).getByText("This key can't use it")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /pick another model/i })).toHaveAttribute("aria-expanded", "true");
    // The key is untouched — a model error is never a disable.
    expect(screen.queryByRole("heading", { level: 1, name: "Settings" })).not.toBeInTheDocument();
  });

  it("falls back to the generic line when the server sends no reason (a pre-m1-10 coach)", async () => {
    coachMock({ key: enabledKey, chat: () => ({ status: 502, body: { error: { code: "provider_error", message: "upstream" } } }) });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    fireEvent.change(await screen.findByLabelText(/message coach/i), { target: { value: "hint please" } });
    fireEvent.click(screen.getByRole("button", { name: /^send$/i }));
    expect(await screen.findByText(/The coach couldn’t reply just now/i)).toBeInTheDocument();
  });

  it("shows F15b's copy when the key itself has been turned off", async () => {
    coachMock({ key: { keys: [{ provider: "openai", masked_key: "sk-...1234", default_model: "gpt-4o-mini", enabled: false }], connected: true } });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    expect(await screen.findByText("Your coach key is turned off")).toBeInTheDocument();
    expect(screen.getByText("Re-enable your provider key in Settings to turn the coach back on.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /add your key in settings/i })).toBeInTheDocument();
  });

  it("reflects the current page in the context chip", async () => {
    coachMock({ key: enabledKey });
    renderApp("/xlearn/dsa/concept/sliding-window");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    expect(await screen.findByText(/Concept — Sliding Window/i)).toBeInTheDocument();
  });
});

describe("Coach page context (m1-03: <course>:<ctx>)", () => {
  afterEach(restoreFetch);

  /** threadContextAt opens the coach at a path and returns the context its thread read
   *  asked for (GET /coach/thread?context=…), plus the chip text. */
  async function threadContextAt(path: string, paths?: unknown): Promise<{ context: string; chip: string }> {
    let context = "";
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (paths && url.endsWith("/api/paths")) return { status: 200, body: paths };
      if (url.endsWith("/api/coach/key")) return { status: 200, body: enabledKey };
      if (url.includes("/api/coach/thread")) {
        context = new URL(url, "http://x").searchParams.get("context") ?? "";
        return { status: 200, body: { messages: [] } };
      }
      return { status: 404 };
    });
    const { unmount } = renderApp(path);
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    await waitFor(() => expect(context).not.toBe(""));
    const chip = document.querySelector(".xl-coach__ctx")?.textContent ?? "";
    unmount();
    return { context, chip };
  }

  it.each([
    ["/xlearn/dsa", "dsa:roadmap", "Roadmap"],
    ["/xlearn/dsa/dashboard", "dsa:dashboard", "Today"],
    ["/xlearn/dsa/revision", "dsa:revision", "Revision"],
    ["/xlearn/dsa/mistakes", "dsa:mistakes", "Mistakes"],
    ["/xlearn/dsa/mock", "dsa:mock", "Mock interview"],
    ["/xlearn/dsa/progress", "dsa:progress", "Progress"],
    ["/xlearn/dsa/week/3", "dsa:week:3", "Week 3"],
    ["/xlearn/dsa/concept/two-pointers", "dsa:concept:two-pointers", "Concept — Two Pointers"],
    // Problems are global ids: unchanged.
    ["/xlearn/dsa/problem/16", "problem:16", "Problem 16"],
    // Account-wide contexts are unchanged.
    ["/xlearn", "catalog", "Catalog"],
    ["/xlearn/settings", "settings", "Settings"],
    // Not a course (NotFound in the plain frame): general.
    ["/xlearn/nope", "general", "xLearn"],
    ["/xlearn/dsa/bogus", "general", "xLearn"],
  ])("%s → %s", async (path, want, chip) => {
    const got = await threadContextAt(path);
    expect(got.context).toBe(want);
    expect(got.chip).toContain(`Context · ${chip}`);
  });

  it("gives a second course its own keys (week 3 of two courses are separate threads)", async () => {
    const both = catalog(DSA_PATH, ZZ_FIXTURE_PATH);
    expect((await threadContextAt("/xlearn/zz-fixture/week/3", both)).context).toBe("zz-fixture:week:3");
    expect((await threadContextAt("/xlearn/dsa/week/3", both)).context).toBe("dsa:week:3");
    expect((await threadContextAt("/xlearn/zz-fixture/dashboard", both)).context).toBe("zz-fixture:dashboard");
  });

  it("uses the general context on a coming-soon course's teaser", async () => {
    const got = await threadContextAt("/xlearn/zz-soon/week/3", catalog(DSA_PATH, ZZ_SOON_PATH));
    expect(got.context).toBe("general");
  });
});

// ---------------------------------------------------------------------------------------
// m1-07 — AB01 F1–F10: the server's mode gate, the D27 assist confirm, fail-closed notices,
// the mock lock, the L18 429s, the cut-short marker, provider-limited, and the naming.
// ---------------------------------------------------------------------------------------

const COACH = "Your AI coach (your key)";
const TRUNCATION_NOTE = "⚠️ This reply was cut off at the length limit — ask me to continue.";
const CAPPED = "Coach used on this attempt · capped at Assisted";
const ASSIST_AT = "2026-10-01T10:00:00Z";

const PROBLEM_16 = { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "", leetcode_url: "", neetcode_url: "", is_reinforcement: false };
const STATEMENT_16 = { stage: "attempt", kind: "summary", order: 1, body_md: "Find all unique triplets that sum to zero.", code: "" };

/** problemAgg is GET /problems/16 mid-attempt, with or without a recorded coach assist. */
function problemAgg(coachAssistAt: string | null = null) {
  return {
    problem: PROBLEM_16,
    sections: [STATEMENT_16],
    state: {
      problemId: "16",
      status: "attempting",
      stageReached: "attempt",
      unlockedStages: ["attempt"],
      currentTouch: 0,
      lastOutcome: null,
      firstSolvedAt: null,
      revealedEarly: false,
      timer: null,
      coachAssistAt,
    },
  };
}

const attemptGate = (coachAssistAt: string | null = null) => ({ mode: "attempt", attempt: { attemptId: "att-1", coachAssistAt } });
const reviewGate = () => ({ mode: "review" });

type ChatBody = Record<string, unknown>;

/** gateMock wires a coach-enabled account: /me, the key, problem 16's aggregate, a thread
 *  with the given gate (omitted when `gate` returns undefined), and a scripted chat — the
 *  n-th POST gets the n-th response (the last one repeats). */
function gateMock(opts: { gate?: () => unknown; thread?: () => unknown[]; problem?: () => unknown; chat?: Array<(body: ChatBody) => Response> }) {
  const calls: ChatBody[] = [];
  let problemGets = 0;
  installFetchMock((url, init) => {
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (url.endsWith("/api/coach/key")) return { status: 200, body: enabledKey };
    if (url.includes("/api/coach/thread")) {
      const gate = opts.gate?.();
      return { status: 200, body: { messages: opts.thread?.() ?? [], ...(gate ? { gate } : {}) } };
    }
    if (url.endsWith("/api/coach/chat")) {
      const body = (init?.body ? JSON.parse(String(init.body)) : {}) as ChatBody;
      calls.push(body);
      const script = opts.chat ?? [];
      const next = script[calls.length - 1] ?? script[script.length - 1];
      return next ? next(body) : { status: 500 };
    }
    if (/\/api\/problems\/16$/.test(url)) {
      problemGets++;
      return { status: 200, body: opts.problem?.() ?? problemAgg() };
    }
    return { status: 404 };
  });
  return { calls, problemGets: () => problemGets };
}

/** reply is a successful SSE chat response stamped with the gateway's X-Coach-Mode. */
const reply = (mode: string, text = "What repeats?") => () =>
  sseResponse([`data: ${JSON.stringify({ delta: text })}\n\n`, `event: done\ndata: {"done":true,"truncated":false}\n\n`], 200, { "X-Coach-Mode": mode });

const confirm409 = () =>
  jsonResponse(409, { error: { code: "assist_confirm_required", message: "confirm first", attemptId: "att-1", problemId: "16" } }, { "X-Coach-Mode": "attempt" });

/** openPanel renders the app at `path` (waiting for problem 16 to load on its page) and
 *  opens the coach. */
async function openPanel(path: string): Promise<HTMLElement> {
  renderApp(path);
  if (path.includes("/problem/16")) await screen.findByRole("heading", { level: 1, name: "3Sum" });
  fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
  return screen.findByRole("complementary", { name: COACH });
}

async function typeAndSend(panel: HTMLElement, text: string) {
  fireEvent.change(await within(panel).findByLabelText(/message coach/i), { target: { value: text } });
  fireEvent.click(within(panel).getByRole("button", { name: /^send$/i }));
}

describe("Coach naming (AB01 naming reference)", () => {
  afterEach(restoreFetch);

  it("names the panel, FAB and close button per the board; Esc closes back to the FAB", async () => {
    coachMock({ key: enabledKey });
    renderApp("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: FAB_NAME }));
    const panel = await screen.findByRole("complementary", { name: COACH });
    expect(panel.querySelector(".xl-coach__title")).toHaveTextContent("Your AI coach (your key)");
    const close = within(panel).getByRole("button", { name: "Close coach" });
    fireEvent.keyDown(close, { key: "Escape" });
    const fab = await screen.findByRole("button", { name: FAB_NAME });
    await waitFor(() => expect(fab).toHaveFocus());
  });
});

describe("Coach mode chip (AB01 F1 / F6 / F7)", () => {
  afterEach(restoreFetch);

  it("F1: attempt mode from gate.mode, with the attempt intro", async () => {
    gateMock({ gate: () => attemptGate() });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    expect(await within(panel).findByText("Attempt · hints only — no pattern or solution")).toBeInTheDocument();
    expect(
      within(panel).getByText(
        "You're mid-attempt on 3Sum, so I'll stick to questions and hints — no pattern names or solution steps until you've concluded.",
      ),
    ).toBeInTheDocument();
    // Not capped yet: no F3 note.
    expect(within(panel).queryByText(CAPPED)).toBeNull();
  });

  it("F6: review mode from gate.mode", async () => {
    gateMock({ gate: reviewGate });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    expect(await within(panel).findByText("Review")).toBeInTheDocument();
    expect(within(panel).queryByText(/Problems you're mid-attempt on/)).toBeNull();
  });

  it("F7: general mode shows its tag and the static off-limits note", async () => {
    gateMock({ gate: () => ({ mode: "general" }) });
    const panel = await openPanel("/xlearn/dsa/week/2");
    expect(await within(panel).findByText("General")).toBeInTheDocument();
    expect(
      within(panel).getByText("Problems you're mid-attempt on stay off-limits in general chat. Ask about them on their own page."),
    ).toBeInTheDocument();
  });

  it("shows no chip when the thread carries no gate, then the mode from X-Coach-Mode", async () => {
    gateMock({ chat: [reply("review")] });
    const panel = await openPanel("/xlearn/dsa/dashboard");
    await within(panel).findByText(/Ask me anything about/);
    expect(panel.querySelector(".xl-coach__mode")).toBeNull();
    await typeAndSend(panel, "how am I doing?");
    expect(await within(panel).findByText("Review")).toBeInTheDocument();
  });
});

describe("Coach D27 assist (AB01 F2 / F3)", () => {
  afterEach(restoreFetch);

  it("F2 → F3: a 409 asks first; Ask re-sends the same message with assist_ack; the HUD chip appears", async () => {
    let acked = false;
    const m = gateMock({
      gate: () => attemptGate(acked ? ASSIST_AT : null),
      problem: () => problemAgg(acked ? ASSIST_AT : null),
      chat: [
        confirm409,
        (body) => {
          acked = body.assist_ack === "att-1";
          return reply("attempt")();
        },
      ],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    const draft = "Is a hash set enough here, or should I sort first?";
    await typeAndSend(panel, draft);

    const card = await within(panel).findByRole("alertdialog", { name: "Ask the coach about this problem?" });
    expect(card).toHaveAccessibleDescription("Using any AI during a counted attempt caps it at Assisted.");
    expect(within(card).getByText("Only this attempt on #16 3Sum is affected.")).toBeInTheDocument();
    expect(within(card).getByRole("button", { name: "Cancel" })).toHaveFocus();
    // Tab moves between the card's two buttons.
    fireEvent.keyDown(within(card).getByRole("button", { name: "Cancel" }), { key: "Tab" });
    expect(within(card).getByRole("button", { name: "Ask — cap at Assisted" })).toHaveFocus();
    // Nothing was sent: the draft waits in the disabled composer, Send is held.
    const input = within(panel).getByLabelText(/message coach/i);
    expect(input).toBeDisabled();
    expect(input).toHaveValue(draft);
    expect(within(panel).getByRole("button", { name: /^send$/i })).toHaveAttribute("aria-disabled", "true");
    expect(panel.querySelector(".xl-coach__msg--me")).toBeNull();
    expect(m.calls[0]!.assist_ack).toBeUndefined();

    const before = m.problemGets();
    fireEvent.click(within(card).getByRole("button", { name: "Ask — cap at Assisted" }));
    await waitFor(() => expect(m.calls).toHaveLength(2));
    expect(m.calls[1]).toMatchObject({ message: draft, assist_ack: "att-1", context: "problem:16" });

    // F3: the panel note, and the Problem HUD chip once the invalidated aggregate refetches.
    expect(await within(panel).findByRole("status")).toHaveTextContent(CAPPED);
    await waitFor(() => expect(m.problemGets()).toBeGreaterThan(before));
    expect(await screen.findByText("Coach used · capped at Assisted")).toBeInTheDocument();
    expect(within(panel).queryByRole("alertdialog")).toBeNull();
  });

  it("F2: Esc is Cancel — the draft stays, nothing is recorded, and Enter never confirms", async () => {
    const m = gateMock({ gate: () => attemptGate(), chat: [confirm409] });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "hint?");
    const card = await within(panel).findByRole("alertdialog");
    const input = within(panel).getByLabelText(/message coach/i);
    // Enter in the composer while the card shows never confirms (or re-sends).
    fireEvent.keyDown(input, { key: "Enter" });
    expect(m.calls).toHaveLength(1);

    fireEvent.keyDown(within(card).getByRole("button", { name: "Cancel" }), { key: "Escape" });
    await waitFor(() => expect(within(panel).queryByRole("alertdialog")).toBeNull());
    // Esc cancelled the card; it didn't close the panel.
    expect(screen.getByRole("complementary", { name: COACH })).toBeInTheDocument();
    expect(input).toBeEnabled();
    expect(input).toHaveValue("hint?");
    expect(m.calls).toHaveLength(1);
    expect(within(panel).queryByText(CAPPED)).toBeNull();
  });

  it("F3: shows the recorded note on load when the attempt already carries coachAssistAt", async () => {
    gateMock({ gate: () => attemptGate(ASSIST_AT), problem: () => problemAgg(ASSIST_AT) });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    expect(await within(panel).findByRole("status")).toHaveTextContent(CAPPED);
  });
});

describe("Coach fail-closed (AB01 F4)", () => {
  afterEach(restoreFetch);

  it("a · assist_unavailable: nothing sent, draft kept; Try again re-sends with the ack", async () => {
    const m = gateMock({
      gate: () => attemptGate(),
      chat: [confirm409, () => jsonResponse(503, { error: { code: "assist_unavailable", message: "practice down" } }, { "X-Coach-Mode": "attempt" }), reply("attempt")],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "Is a hash set enough?");
    fireEvent.click(await within(panel).findByRole("button", { name: "Ask — cap at Assisted" }));

    const alert = await within(panel).findByRole("alert");
    expect(alert).toHaveTextContent("We couldn't record coach use on your attempt, so nothing was sent. Try again.");
    expect(within(panel).getByText("Your message wasn't sent. It's still here.")).toBeInTheDocument();
    expect(within(panel).getByLabelText(/message coach/i)).toHaveValue("Is a hash set enough?");
    // The record failed, so the attempt isn't capped.
    expect(within(panel).queryByText(CAPPED)).toBeNull();

    fireEvent.click(within(alert).getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(m.calls).toHaveLength(3));
    expect(m.calls[2]).toMatchObject({ message: "Is a hash set enough?", assist_ack: "att-1" });
    expect(await within(panel).findByText(CAPPED)).toBeInTheDocument();
  });

  it("b · coach_state_unavailable: its own copy; Try again re-sends the kept draft", async () => {
    const m = gateMock({
      chat: [() => jsonResponse(503, { error: { code: "coach_state_unavailable", message: "lookup failed" } }), reply("review")],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "And the space cost?");
    const alert = await within(panel).findByRole("alert");
    expect(alert).toHaveTextContent("We couldn't check your practice state, so nothing was sent. Try again in a moment.");
    expect(within(panel).getByLabelText(/message coach/i)).toHaveValue("And the space cost?");
    expect(within(panel).getByLabelText(/message coach/i)).toBeEnabled();

    fireEvent.click(within(alert).getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(m.calls).toHaveLength(2));
    expect(m.calls[1]!.message).toBe("And the space cost?");
    expect(m.calls[1]!.assist_ack).toBeUndefined();
  });
});

describe("Coach paused (AB01 F5)", () => {
  afterEach(restoreFetch);

  /** expectPaused checks F5: the Paused chip, the status banner, the disabled composer with
   *  the reason as its placeholder, Send aria-disabled, and focus on Close. */
  async function expectPaused(panel: HTMLElement) {
    expect(await within(panel).findByText("Paused")).toBeInTheDocument();
    expect(within(panel).getByRole("status")).toHaveTextContent("The coach is paused while a revision touch or mock is live. It's back when you finish.");
    const input = within(panel).getByLabelText(/message coach/i);
    expect(input).toBeDisabled();
    expect(input).toHaveAttribute("placeholder", "Paused while a mock or touch is live");
    expect(within(panel).getByRole("button", { name: /^send$/i })).toHaveAttribute("aria-disabled", "true");
    await waitFor(() => expect(within(panel).getByRole("button", { name: "Close coach" })).toHaveFocus());
  }

  it("locks from gate.mode = locked on load", async () => {
    gateMock({ gate: () => ({ mode: "locked", reason: "mock" }) });
    const panel = await openPanel("/xlearn/dsa/mock");
    await expectPaused(panel);
    // No intro or suggestions to send while paused.
    expect(panel.querySelector(".xl-coach__sugg")).toBeNull();
  });

  it("locks on a 409 coach_paused; nothing was sent", async () => {
    const m = gateMock({
      gate: reviewGate,
      chat: [() => jsonResponse(409, { error: { code: "coach_paused", message: "paused", reason: "mock" } }, { "X-Coach-Mode": "locked" })],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "quick question");
    await expectPaused(panel);
    expect(panel.querySelector(".xl-coach__msg--me")).toBeNull();
    expect(m.calls).toHaveLength(1);
  });
});

describe("Coach L18 limits (AB01 F8)", () => {
  afterEach(() => {
    vi.useRealTimers();
    restoreFetch();
  });

  const fakeClock = () => vi.useFakeTimers({ shouldAdvanceTime: true, toFake: ["setTimeout", "clearTimeout", "setInterval", "clearInterval", "Date"] });
  const limited = (code: string, retryAfter: string) => () => jsonResponse(429, { error: { code, message: "limited" } }, { "Retry-After": retryAfter });

  it("F8a: coach_rate_limited counts down from Retry-After and clears at 0; the draft stays", async () => {
    fakeClock();
    gateMock({ gate: reviewGate, chat: [limited("coach_rate_limited", "12")] });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "What if the array is already sorted?");

    const note = await within(panel).findByRole("status");
    expect(note).toHaveTextContent("You're sending quickly — try again in 12 s");
    const send = within(panel).getByRole("button", { name: /^send$/i });
    expect(send).toHaveAttribute("aria-disabled", "true");
    const input = within(panel).getByLabelText(/message coach/i);
    expect(input).toHaveValue("What if the array is already sorted?");
    // The number updates without re-announcing the region.
    expect(within(note).getByText("12 s")).toHaveAttribute("aria-live", "off");

    // Flush the countdown's effect (its interval) before moving the clock. The fake clock
    // also advances with real time (shouldAdvanceTime), so a slow runner may already be
    // a second further along: the number must have dropped, to 11 or 10.
    await act(async () => {});
    act(() => void vi.advanceTimersByTime(1000));
    await waitFor(() => expect(note).toHaveTextContent(/try again in 1[01] s/));
    act(() => void vi.advanceTimersByTime(11_000));
    await waitFor(() => expect(within(panel).queryByRole("status")).toBeNull());
    expect(send).not.toHaveAttribute("aria-disabled");
    expect(input).toHaveValue("What if the array is already sorted?");
  });

  it("F8b: coach_busy holds Send until Retry-After has passed", async () => {
    fakeClock();
    gateMock({ gate: reviewGate, chat: [limited("coach_busy", "5")] });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "And the space cost?");

    expect(await within(panel).findByRole("status")).toHaveTextContent("One reply at a time — wait for the current reply");
    const send = within(panel).getByRole("button", { name: /^send$/i });
    expect(send).toHaveAttribute("aria-disabled", "true");
    act(() => void vi.advanceTimersByTime(5_000));
    await waitFor(() => expect(within(panel).queryByRole("status")).toBeNull());
    expect(send).not.toHaveAttribute("aria-disabled");
    expect(within(panel).getByLabelText(/message coach/i)).toHaveValue("And the space cost?");
  });

  it("F8c: coach_daily_cap shows the UTC reset on the learner's clock and holds the input until then", async () => {
    fakeClock();
    vi.setSystemTime(new Date("2026-10-01T18:30:00Z"));
    gateMock({ gate: reviewGate, chat: [limited("coach_daily_cap", "19800")] }); // 5.5 h to 00:00 UTC
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "And the space cost?");

    // now + Retry-After is the next UTC midnight, formatted in the browser's locale.
    const local = new Date("2026-10-02T00:00:00Z").toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
    const note = await within(panel).findByRole("status");
    expect(note).toHaveTextContent(`You've reached today's 300-message limit on your key. It resets at 00:00 UTC (${local} your time).`);
    const input = within(panel).getByLabelText(/message coach/i);
    expect(input).toBeDisabled();
    expect(input).toHaveAttribute("placeholder", `Back at ${local} your time`);
    expect(within(panel).getByRole("button", { name: /^send$/i })).toHaveAttribute("aria-disabled", "true");

    act(() => void vi.advanceTimersByTime(19_800_000));
    await waitFor(() => expect(within(panel).queryByRole("status")).toBeNull());
    expect(input).toBeEnabled();
    // The draft comes back once the hold clears.
    expect(input).toHaveValue("And the space cost?");
  });
});

describe("Coach cut-short reply (AB01 F9)", () => {
  afterEach(restoreFetch);

  it("marks a truncated reply, strips the persisted note, and Continue sends exactly one fixed turn", async () => {
    let persisted: unknown[] = [];
    const m = gateMock({
      gate: reviewGate,
      thread: () => persisted,
      chat: [
        (body) => {
          persisted = [
            { role: "user", content: body.message },
            { role: "assistant", content: `3. On memory: every word is stored once under its key, and\n\n${TRUNCATION_NOTE}` },
          ];
          return sseResponse(
            [
              `data: ${JSON.stringify({ delta: "3. On memory: every word is stored once under its key, and" })}\n\n`,
              `data: ${JSON.stringify({ delta: `\n\n${TRUNCATION_NOTE}` })}\n\n`,
              `event: done\ndata: {"done":true,"truncated":true}\n\n`,
            ],
            200,
            { "X-Coach-Mode": "review" },
          );
        },
        reply("review", "Picking up: the map holds one entry per distinct key."),
      ],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "Review my submission line by line?");

    expect(await within(panel).findByText("Reply cut short (length limit)")).toBeInTheDocument();
    expect(within(panel).queryByText(/cut off at the length limit/)).toBeNull();
    // Continue is a real button; it doesn't touch the learner's own draft.
    const input = within(panel).getByLabelText(/message coach/i);
    fireEvent.change(input, { target: { value: "my own draft" } });
    fireEvent.click(within(panel).getByRole("button", { name: "Continue" }));
    await waitFor(() => expect(m.calls).toHaveLength(2));
    expect(m.calls[1]!.message).toBe("Continue from where you stopped.");
    expect(m.calls[1]!.context).toBe("problem:16");
    expect(input).toHaveValue("my own draft");
    await new Promise((r) => setTimeout(r, 50));
    expect(m.calls).toHaveLength(2);
  });

  it("renders the same marker for a history reply that ends with the persisted note", async () => {
    gateMock({
      gate: reviewGate,
      thread: () => [
        { role: "user", content: "Review my submission line by line?" },
        { role: "assistant", content: `Here's a line-by-line pass.\n\n${TRUNCATION_NOTE}` },
      ],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    expect(await within(panel).findByText("Reply cut short (length limit)")).toBeInTheDocument();
    expect(within(panel).getByText("Here's a line-by-line pass.")).toBeInTheDocument();
    expect(within(panel).queryByText(/cut off at the length limit/)).toBeNull();
    expect(within(panel).getByRole("button", { name: "Continue" })).toBeInTheDocument();
  });
});

describe("Coach provider limited (AB01 F10)", () => {
  afterEach(restoreFetch);

  it("keeps the key on, says so, and Retry re-sends the refused turn once", async () => {
    let persisted: unknown[] = [];
    const m = gateMock({
      gate: reviewGate,
      thread: () => persisted,
      chat: [
        () => jsonResponse(429, { error: { code: "provider_limited", message: "out of credit", reason: "quota" } }),
        (body) => {
          persisted = [
            { role: "user", content: body.message },
            { role: "assistant", content: "What do you keep around for the whole pass?" },
          ];
          return reply("review", "What do you keep around for the whole pass?")();
        },
      ],
    });
    const panel = await openPanel("/xlearn/dsa/problem/16");
    await typeAndSend(panel, "What would change for a stream of numbers?");

    const alert = await within(panel).findByRole("alert");
    expect(alert).toHaveTextContent("Your provider says this key is out of credit or rate-limited. Top up or wait, then retry.");
    expect(within(panel).getByText("Your OpenAI key is still on.")).toBeInTheDocument();
    // Distinct from the L18 notices: no countdown, nothing held.
    expect(within(panel).queryByRole("status")).toBeNull();
    expect(within(panel).getByRole("button", { name: /^send$/i })).not.toHaveAttribute("aria-disabled");

    fireEvent.click(within(alert).getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(m.calls).toHaveLength(2));
    expect(m.calls[1]!.message).toBe("What would change for a stream of numbers?");
    expect(await within(panel).findByText("What do you keep around for the whole pass?")).toBeInTheDocument();
    expect(within(panel).getAllByText("What would change for a stream of numbers?")).toHaveLength(1);
    expect(within(panel).queryByRole("alert")).toBeNull();
  });
});
