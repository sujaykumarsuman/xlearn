import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, describe, expect, it } from "vitest";
import type { CourseViewBand } from "../lib/curriculum";
import { bandDuration, bandFor, formatNextReview, nextReviewDate } from "../lib/revision";
import type { DueItem } from "../lib/revision";
import { routes } from "../router";
import { DSA_PATH, DSA_VIEW, catalog } from "../test/courses";
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
const later = new Date(Date.now() + 9 * 24 * 3600_000).toISOString();

/** DSA's revision bands as the course view carries them (m1-06): re-solve on a 20:00
 *  timer, mock conditions at L4–5. */
const DSA_BANDS: CourseViewBand[] = [
  { levels: [1, 2, 3], format: "resolve", label: "Re-solve", timer_s: 1200, est_minutes: 20, mock_mode: false },
  { levels: [4, 5], format: "resolve", label: "Re-solve", timer_s: 1200, est_minutes: 20, mock_mode: true },
];

/** A GET /paths body whose DSA course view carries the given revision bands. */
function catalogWithBands(bands: CourseViewBand[]) {
  return catalog({ ...DSA_PATH, course: { ...DSA_VIEW, revision: { bands } } });
}

// A due item carries no pattern: the gateway withholds it while the touch is due (m1-06).
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
    problem: { id: "3", title: "Two Sum", difficulty: "easy", week_n: 1 },
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
      problem: { id: "16", title: "3Sum", difficulty: "med", week_n: 2 },
    }),
    item({
      itemId: "it-4",
      problemId: "23",
      touchLevel: 4,
      dayLabel: "Day 21",
      mockMode: true,
      problem: { id: "23", title: "Largest Rectangle in Histogram", difficulty: "hard", week_n: 4 },
    }),
    // Upcoming, solved and not live: the payload keeps its pattern.
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
  dueCount: 3,
};

/** The score response: `problem` is composed after the touch concluded, so it carries the
 *  pattern (the AB03-F3/F4 reveal). */
function scoreBody(over: Partial<Record<string, unknown>> = {}) {
  return {
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
    problem: { id: "3", title: "Two Sum", difficulty: "easy", pattern: "Complement lookup", week_n: 1 },
    ...over,
  };
}

interface MockOpts {
  queue?: unknown;
  bands?: CourseViewBand[] | null; // null → the default catalog (no bands)
  score?: (body: Record<string, unknown>) => unknown;
}

function installRevisionMock({ queue = QUEUE, bands = DSA_BANDS, score }: MockOpts = {}) {
  return installFetchMock((url, init) => {
    if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
    if (bands && new URL(url, "http://x").pathname.endsWith("/api/paths")) return { status: 200, body: catalogWithBands(bands) };
    if (score && url.includes("/api/revision/") && url.endsWith("/score") && init?.method === "POST") {
      return { status: 200, body: score(JSON.parse(String(init?.body ?? "{}"))) };
    }
    if (url.includes("/api/paths/dsa/revision/due")) return { status: 200, body: queue };
    return { status: 404 };
  });
}

/** The card (queue row) holding a problem title. */
function cardOf(title: string): HTMLElement {
  const el = screen.getByText(title).closest<HTMLElement>(".ds-card, .xl-panel");
  if (!el) throw new Error(`no card for ${title}`);
  return el;
}

/** The format badges' text inside an element, whitespace-normalised. */
function fmtTexts(root: HTMLElement): string[] {
  return Array.from(root.querySelectorAll(".rv-fmt")).map((e) => (e.textContent ?? "").replace(/\s+/g, " ").trim());
}

/** The role="status" result region around a result heading. */
function resultRegion(el: HTMLElement): HTMLElement {
  const region = el.closest<HTMLElement>('[role="status"]');
  if (!region) throw new Error("result is not in a status region");
  return region;
}

async function startFirstReview() {
  await userEvent.click(await screen.findByRole("button", { name: /Start next review/ }));
}

describe("Revision queue", () => {
  afterEach(restoreFetch);

  it("renders the prioritised due queue grouped by touch day, with an upcoming tail", async () => {
    installRevisionMock();
    renderApp("/xlearn/dsa/revision");

    expect(await screen.findByRole("heading", { level: 1, name: "Revision queue" })).toBeInTheDocument();
    // "N due today" from dueCount (await the query resolving).
    expect(await screen.findByText(/3 due today/)).toBeInTheDocument();
    // Grouped by touch day; the mock band's group says so.
    expect(screen.getByRole("heading", { name: /Due · Day 1/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: /Due · Day 7/ })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Due · Day 21 · mock conditions" })).toBeInTheDocument();
    // Enriched problem titles.
    expect(screen.getByText("Two Sum")).toBeInTheDocument();
    expect(screen.getByText("3Sum")).toBeInTheDocument();
    // The not-due item is in "Coming up", not the due groups.
    expect(screen.getByRole("heading", { name: "Coming up" })).toBeInTheDocument();
    expect(screen.getByText("Valid Anagram")).toBeInTheDocument();
    expect(screen.getByText("Not due yet")).toBeInTheDocument();
  });

  it("renders a format badge per card from the course band for its touch level (AB03-F1)", async () => {
    installRevisionMock();
    renderApp("/xlearn/dsa/revision");
    await screen.findByText("Two Sum");

    expect(fmtTexts(cardOf("Two Sum"))).toEqual(["Re-solve · 20:00"]);
    expect(fmtTexts(cardOf("3Sum"))).toEqual(["Re-solve · 20:00"]);
    // L4: the mock band, then v1's violet mock badge AFTER the format badge.
    const hard = cardOf("Largest Rectangle in Histogram");
    expect(fmtTexts(hard)).toEqual(["Re-solve · mock conditions · 20:00"]);
    const badges = hard.querySelector(".rv-line__badges")!;
    expect(Array.from(badges.children).map((c) => c.className)).toEqual([
      "xl-lock",
      "xl-tag rv-fmt",
      "ds-badge ds-badge--violet",
    ]);
    expect(within(hard).getByText("mock")).toBeInTheDocument();
    // The timer is bold (mono) inside the badge.
    expect(hard.querySelector(".rv-fmt b")?.textContent).toBe("20:00");
  });

  it("renders the Recall variant only when a band says so (fixture band, AB03 rule strip)", async () => {
    installRevisionMock({
      bands: [
        { levels: [1, 2, 3], format: "recall", label: "Recall", est_minutes: 5, mock_mode: false },
        DSA_BANDS[1]!,
      ],
    });
    renderApp("/xlearn/dsa/revision");
    await screen.findByText("Two Sum");

    expect(fmtTexts(cardOf("Two Sum"))).toEqual(["Recall · ~5 min"]);
    expect(fmtTexts(cardOf("Largest Rectangle in Histogram"))).toEqual(["Re-solve · mock conditions · 20:00"]);
  });

  it("shows no format badge without a band for the level", async () => {
    installRevisionMock({ bands: null }); // the default catalog: no revision block
    renderApp("/xlearn/dsa/revision");
    await screen.findByText("Two Sum");

    expect(document.querySelectorAll(".rv-fmt")).toHaveLength(0);
    // v1's mock badge still marks the mock-mode row.
    expect(within(cardOf("Largest Rectangle in Histogram")).getByText("mock")).toBeInTheDocument();
  });

  it("withholds the pattern on due cards: a lock, never a chip (AB03-F1)", async () => {
    installRevisionMock();
    renderApp("/xlearn/dsa/revision");
    await screen.findByText("Two Sum");

    for (const title of ["Two Sum", "3Sum", "Largest Rectangle in Histogram"]) {
      const card = cardOf(title);
      expect(within(card).getByText("Pattern hidden while due")).toBeInTheDocument();
      expect(card.querySelector(".xl-pat")).toBeNull();
    }
    // An upcoming, solved (not live) row keeps its chip.
    const upcoming = cardOf("Valid Anagram");
    expect(within(upcoming).getByText("HashMap")).toBeInTheDocument();
    expect(upcoming.querySelector(".xl-pat")).not.toBeNull();
    expect(within(upcoming).queryByText("Pattern hidden while due")).toBeNull();
  });

  it("keeps the lock on a never-solved upcoming row and shows neither for an unresolved problem", async () => {
    installRevisionMock({
      queue: {
        items: [
          item({ itemId: "it-7", problemId: "7", due: false, dueDate: future, problem: { id: "7", title: "Top K Frequent Elements", difficulty: "med", week_n: 2 } }),
          item({ itemId: "it-9", problemId: "99", due: false, dueDate: later, problem: null }),
        ],
        dueCount: 0,
      },
    });
    renderApp("/xlearn/dsa/revision");
    await screen.findByText("Top K Frequent Elements");

    expect(within(cardOf("Top K Frequent Elements")).getByText("Pattern hidden while due")).toBeInTheDocument();
    const bare = cardOf("Problem 99");
    expect(within(bare).queryByText("Pattern hidden while due")).toBeNull();
    expect(bare.querySelector(".xl-pat")).toBeNull();
  });

  it("runs the re-solve on the band's timer, pattern still hidden (AB03-F2)", async () => {
    installRevisionMock({
      bands: [{ levels: [1, 2, 3], format: "resolve", label: "Re-solve", timer_s: 900, mock_mode: false }, DSA_BANDS[1]!],
    });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();

    const timer = await screen.findByRole("timer");
    expect(timer).toHaveTextContent("15:00");
    expect(timer).toHaveAttribute("aria-label", "15:00 remaining");
    expect(screen.getByText("solved before the 15:00 timer")).toBeInTheDocument();
    // The active card's header keeps the lock; the badge shows the band's timer.
    const active = cardOf("Two Sum");
    expect(within(active).getByText("Pattern hidden while due")).toBeInTheDocument();
    expect(fmtTexts(active)).toEqual(["Re-solve · 15:00"]);
    // v1's three criteria tiles, as toggle buttons.
    expect(screen.getByRole("button", { name: /Named the pattern/ })).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByRole("button", { name: /Solved within the timer/ })).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByRole("button", { name: /Stated the complexity/ })).toBeInTheDocument();
  });

  it("falls back to v1's 20:00 timer without a band", async () => {
    installRevisionMock({ bands: null });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();

    expect(await screen.findByRole("timer")).toHaveTextContent("20:00");
  });

  it("re-solving submits the auto-score inputs and shows the pass → advance result", async () => {
    const fetchMock = installRevisionMock({ score: () => scoreBody() });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();

    // The re-solve panel appears on the band's 20:00 timer.
    expect(await screen.findByRole("timer")).toHaveTextContent("20:00");
    await userEvent.click(screen.getByRole("button", { name: /Submit re-solve for auto-score/ }));

    // The auto-score result panel: pass advances to Day 3.
    expect(await screen.findByText(/Passed — advances to Day 3/)).toBeInTheDocument();
    // The score endpoint was actually called with a POST.
    expect(
      fetchMock.mock.calls.some(([u, i]) => String(u).includes("/api/v1/revision/it-1/score") && (i as RequestInit)?.method === "POST"),
    ).toBe(true);
  });

  it("a scored pass reveals the pattern in the header and the result (AB03-F3)", async () => {
    installRevisionMock({ score: () => scoreBody() });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();
    expect(screen.queryByText("Complement lookup")).toBeNull();
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    const status = resultRegion(await screen.findByText(/Passed — advances to Day 3/));
    // Reveal row inside the result, and the chip replaces the lock in the header.
    expect(within(status).getByText("Complement lookup")).toBeInTheDocument();
    expect(within(status).getByText(/^Pattern/)).toBeInTheDocument();
    const card = cardOf("Two Sum");
    expect(card.querySelectorAll(".xl-pat")).toHaveLength(2);
    expect(within(card).queryByText("Pattern hidden while due")).toBeNull();
    // A pass opens no mistake entry; focus moves to "Next review".
    expect(within(status).queryByText(/A mistake entry is open/)).toBeNull();
    expect(screen.getByRole("button", { name: "Next review" })).toHaveFocus();
  });

  it("a pass advancing into Day 21 flags mock conditions (from the next touch, not the scored one)", async () => {
    // Scored a Day-7 touch (not mock); it advances to Day 21 (mock). mockMode is the
    // scored touch's flag (false) — the panel must use the NEXT level instead.
    installRevisionMock({ score: () => scoreBody({ touchLevel: 3, nextTouchLevel: 4, nextDayLabel: "Day 21" }) });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    expect(await screen.findByText(/Next review Day 21 \(mock conditions\)/)).toBeInTheDocument();
  });

  it("submits a failing pattern time when the learner never affirms 'Named the pattern'", async () => {
    let sent: Record<string, unknown> | null = null;
    installRevisionMock({
      score: (body) => {
        sent = body;
        return scoreBody({ autoPass: false, status: "failed", reset: true, nextTouchLevel: 1, nextDayLabel: "Day 1" });
      },
    });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();
    // Submit WITHOUT tapping "Named the pattern": the payload must carry a value that
    // fails the < 120s check, not the small elapsed time.
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    await screen.findByText(/reset to Day 1/);
    expect(sent).not.toBeNull();
    expect((sent as unknown as { namedPatternSecs: number }).namedPatternSecs).toBe(120);
  });

  it("a miss resets to Day 1, reveals the pattern and links the open mistake entry (AB03-F4)", async () => {
    installRevisionMock({
      score: () => scoreBody({ autoPass: false, status: "failed", reset: true, nextTouchLevel: 1, nextDayLabel: "Day 1" }),
    });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    const status = resultRegion(await screen.findByText(/Missed — reset to Day 1/));
    expect(within(status).getByText("Complement lookup")).toBeInTheDocument();
    const link = within(status).getByRole("link", { name: /A mistake entry is open/ });
    expect(link).toHaveAttribute("href", "/xlearn/dsa/mistakes");
  });

  it("shows no reveal chip when the result carries no pattern (still live)", async () => {
    installRevisionMock({
      score: () => scoreBody({ problem: { id: "3", title: "Two Sum", difficulty: "easy", week_n: 1 } }),
    });
    renderApp("/xlearn/dsa/revision");
    await startFirstReview();
    await userEvent.click(await screen.findByRole("button", { name: /Submit re-solve for auto-score/ }));

    const status = resultRegion(await screen.findByText(/Passed — advances to Day 3/));
    expect(status.querySelector(".xl-pat")).toBeNull();
    expect(within(cardOf("Two Sum")).getByText("Pattern hidden while due")).toBeInTheDocument();
  });

  it("shows the empty state without a date when nothing is upcoming (AB03-F6)", async () => {
    installRevisionMock({ queue: { items: [], dueCount: 0 } });
    renderApp("/xlearn/dsa/revision");

    const heading = await screen.findByText("Nothing due today.");
    const status = heading.closest<HTMLElement>('[role="status"]')!;
    expect(status).not.toBeNull();
    expect(status).not.toHaveTextContent(/Next review:/);
    expect(within(status).getByRole("link", { name: "Roadmap" })).toHaveAttribute("href", "/xlearn/dsa");
  });

  it("appends the next review date from the payload's earliest upcoming touch (AB03-F6)", async () => {
    installRevisionMock({
      queue: {
        items: [
          item({ itemId: "it-8", problemId: "8", due: false, dueDate: later, problem: { id: "8", title: "Later One", difficulty: "easy", pattern: "Stack", week_n: 3 } }),
          item({ itemId: "it-2", problemId: "2", due: false, dueDate: future, problem: { id: "2", title: "Valid Anagram", difficulty: "easy", pattern: "HashMap", week_n: 1 } }),
        ],
        dueCount: 0,
      },
    });
    renderApp("/xlearn/dsa/revision");

    const heading = await screen.findByText(`Nothing due today. Next review: ${formatNextReview(new Date(future))}.`);
    expect(heading.closest('[role="status"]')).not.toBeNull();
    // The upcoming tail still renders.
    expect(screen.getByRole("heading", { name: "Coming up" })).toBeInTheDocument();
  });
});

describe("revision band helpers", () => {
  const recall: CourseViewBand = { levels: [1, 2, 3], format: "recall", label: "Recall", est_minutes: 5, mock_mode: false };

  it("picks the band covering a level", () => {
    expect(bandFor(DSA_BANDS, 2)).toBe(DSA_BANDS[0]);
    expect(bandFor(DSA_BANDS, 5)).toBe(DSA_BANDS[1]);
    expect(bandFor(DSA_BANDS, 6)).toBeUndefined();
    expect(bandFor(undefined, 1)).toBeUndefined();
  });

  it("renders a timed band as mm:ss and an untimed one as ~N min", () => {
    expect(bandDuration(DSA_BANDS[0]!)).toBe("20:00");
    expect(bandDuration(recall)).toBe("~5 min");
    expect(bandDuration({ ...recall, est_minutes: undefined })).toBe("");
  });

  it("finds the earliest upcoming touch, ignoring due items", () => {
    const items = [
      { due: true, dueDate: past },
      { due: false, dueDate: later },
      { due: false, dueDate: future },
    ] as DueItem[];
    expect(nextReviewDate(items)?.toISOString()).toBe(future);
    expect(nextReviewDate([{ due: true, dueDate: past } as DueItem])).toBeNull();
  });
});
