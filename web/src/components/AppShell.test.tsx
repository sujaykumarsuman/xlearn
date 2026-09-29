import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { RouterProvider, createMemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { routes } from "../router";
import { DSA_PATH, SYSTEM_DESIGN_PATH, ZZ_FIXTURE_PATH, ZZ_SOON_PATH, catalog } from "../test/courses";
import { authedMe, installFetchMock, restoreFetch, type RouteHandler } from "../test/fetchMock";

function renderAt(path: string) {
  const router = createMemoryRouter(routes, { initialEntries: [path], basename: "/xlearn" });
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>,
  );
}

describe("AppShell + routing (authenticated)", () => {
  beforeEach(() => {
    // Authenticated + onboarded so app routes render the shell (not onboarding).
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 200, body: authedMe("dsa") } : { status: 404 }));
  });
  afterEach(restoreFetch);

  it("renders the shell chrome and the routed screen", async () => {
    renderAt("/xlearn/dsa/dashboard");
    expect(await screen.findByRole("navigation")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /roadmap/i })).toBeInTheDocument();
    expect(await screen.findByRole("heading", { level: 1, name: "Today" })).toBeInTheDocument();
  });

  it("marks the active nav item for the current route", async () => {
    renderAt("/xlearn/dsa/revision");
    await screen.findByRole("navigation");
    const active = document.querySelector(".xl-nav__item--active");
    expect(active?.textContent).toContain("Revision");
  });

  it("shows live Practice-loop badge counts (reviews due · open mistakes)", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths/dsa/revision/due")) return { status: 200, body: { items: [], dueCount: 2 } };
      if (url.endsWith("/api/paths/dsa/mistakes"))
        return { status: 200, body: { mistakes: [], openCount: 5, closedCount: 0, closeThreshold: 0, categories: [] } };
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/dashboard");

    const nav = await screen.findByRole("navigation");
    await waitFor(() => {
      expect(within(nav).getByRole("link", { name: /revision/i })).toHaveTextContent("2");
      expect(within(nav).getByRole("link", { name: /mistakes/i })).toHaveTextContent("5");
    });
  });

  it("routes the sidebar brand home", async () => {
    renderAt("/xlearn/dsa/dashboard");
    const brand = await screen.findByRole("link", { name: /xlearn — home/i });
    expect(brand).toHaveAttribute("href", "/xlearn");
  });

  it("renders params-driven screens", async () => {
    // The Problem workspace fetches its BFF aggregate; give it a minimal one.
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.includes("/api/problems/16")) {
        return {
          status: 200,
          body: {
            problem: { id: "16", path_slug: "dsa", week_n: 2, title: "3Sum", difficulty: "med", pattern: "Two Pointers", leetcode_url: "", neetcode_url: "", is_reinforcement: false },
            sections: [{ stage: "attempt", kind: "summary", order: 1, body_md: "Find all triplets that sum to zero.", code: "" }],
            state: { problemId: "16", status: "available", stageReached: "", unlockedStages: ["attempt"], currentTouch: 0, lastOutcome: null, firstSolvedAt: null, revealedEarly: false, timer: null },
          },
        };
      }
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/problem/16");
    expect(await screen.findByRole("heading", { level: 1, name: "3Sum" })).toBeInTheDocument();
  });

  it("opens the top-bar curriculum switcher dropdown (F001)", async () => {
    renderAt("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /switch curriculum/i }));
    expect(screen.getByText("Browse all paths")).toBeInTheDocument();
  });

  it("renders every app route without crashing — curriculum routes get the sidebar, hub routes don't (F001)", async () => {
    const curriculum = [
      "/xlearn/dsa",
      "/xlearn/dsa/dashboard",
      "/xlearn/dsa/week/2",
      "/xlearn/dsa/concept/sliding-window",
      "/xlearn/dsa/problem/16",
      "/xlearn/dsa/revision",
      "/xlearn/dsa/mistakes",
      "/xlearn/dsa/mock",
      "/xlearn/dsa/progress",
    ];
    // Unknown paths — single- or multi-segment — hit the in-shell NotFound: public profiles
    // live under /xlearn/u/<username> (ADR-0025), so a bare /xlearn/<x> is no longer one.
    const hub = ["/xlearn", "/xlearn/settings", "/xlearn/nope", "/xlearn/nope/404"];

    for (const path of curriculum) {
      const { unmount } = renderAt(path);
      expect(await screen.findByRole("navigation")).toBeInTheDocument();
      expect(document.querySelector(".xl-app")).not.toBeNull();
      unmount();
    }
    for (const path of hub) {
      const { unmount } = renderAt(path);
      // The sidebar-less shell: brand lockup present, no sidebar nav.
      expect(await screen.findByRole("link", { name: /all paths/i })).toBeInTheDocument();
      expect(document.querySelector(".xl-app")).not.toBeNull();
      expect(document.querySelector(".xl-side")).toBeNull();
      unmount();
    }
  });
});

describe("auth gating", () => {
  afterEach(restoreFetch);

  it("redirects unauthenticated app-route visits to the standalone /auth screen", async () => {
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 401, body: { error: { code: "unauthenticated" } } } : { status: 404 }));
    renderAt("/xlearn/dsa/dashboard");
    // Auth screen (no app shell) with the OAuth buttons.
    expect(await screen.findByRole("button", { name: /continue with github/i })).toBeInTheDocument();
    expect(document.querySelector(".xl-app")).toBeNull();
  });

  it("redirects an authenticated-but-un-onboarded user from an app route into onboarding", async () => {
    // Authenticated (200) but path_chosen=null → must land in onboarding, not the app.
    installFetchMock((url) => (url.endsWith("/api/me") ? { status: 200, body: authedMe(null) } : { status: 404 }));
    renderAt("/xlearn/dsa/dashboard");
    expect(await screen.findByRole("heading", { name: /pick your path/i })).toBeInTheDocument();
    expect(document.querySelector(".xl-app")).toBeNull();
  });

  it("keeps the user in the app and shows an error when sign-out fails", async () => {
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/auth/logout")) return { status: 502, body: { error: { code: "upstream" } } };
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/dashboard");
    await screen.findByRole("navigation");
    // Open the account menu (button-disclosure) before reaching Sign out.
    fireEvent.click(screen.getByRole("button", { name: /account menu/i }));
    fireEvent.click(screen.getByRole("button", { name: /sign out/i }));
    // A failed logout surfaces an error and does NOT navigate to /auth.
    expect(await screen.findByText(/couldn’t sign out/i)).toBeInTheDocument();
    expect(screen.getByRole("navigation")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /continue with github/i })).not.toBeInTheDocument();
  });
});

describe("course resolution (useCourse, AB02)", () => {
  afterEach(restoreFetch);

  /** withCatalog answers /me and /paths, then defers to `rest`. Returns the fetch mock. */
  function withCatalog(paths: unknown, rest: RouteHandler = () => ({ status: 404 })) {
    return installFetchMock((url, init) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths")) return { status: 200, body: paths };
      return rest(url, init);
    });
  }

  /** The account-wide calls any frame makes (session, catalog, the top bar's coach model
   *  switcher); anything else would be a course data call. */
  const ACCOUNT_CALLS = new Set(["/me", "/paths", "/coach/key"]);

  /** requested lists the API paths fetched (without the /api/v1 prefix). */
  function requested(fn: ReturnType<typeof installFetchMock>): string[] {
    return fn.mock.calls.map(([u]) => String(u).replace(/^.*\/api\/v1/, ""));
  }

  it("active: DSA renders the v1 frame — manifest nav, switcher DSA · title, crumbs", async () => {
    withCatalog(catalog());
    renderAt("/xlearn/dsa/revision");
    const nav = await screen.findByRole("navigation");
    const labels = within(nav).getAllByRole("link").map((a) => a.textContent);
    expect(labels).toEqual(["Today", "Roadmap", "Problems", "Progress", "Revision", "Mistakes", "Mock interview"]);
    expect(within(nav).getByRole("link", { name: "Roadmap" })).toHaveAttribute("href", "/xlearn/dsa");
    expect(nav.querySelector(".xl-nav__cap")?.textContent).toBe("Practice loop");
    const sw = screen.getByRole("button", { name: /switch curriculum/i });
    expect(sw.querySelector(".xl-pathsw__ic")?.textContent).toBe("DSA");
    expect(sw.querySelector(".xl-pathsw__t")?.textContent).toBe("Data Structures & Algorithms");
    expect(document.querySelector(".xl-crumb")?.textContent).toBe("xlearn/dsa/revision");
  });

  it("active: the switcher lists open courses → their dashboards, and coming-soon ones locked", async () => {
    withCatalog(catalog(DSA_PATH, ZZ_FIXTURE_PATH, SYSTEM_DESIGN_PATH));
    renderAt("/xlearn/dsa/dashboard");
    fireEvent.click(await screen.findByRole("button", { name: /switch curriculum/i }));
    const menu = screen.getByRole("menu");
    expect(within(menu).getByRole("menuitem", { name: /Fixture Course/ })).toHaveAttribute("href", "/xlearn/zz-fixture/dashboard");
    expect(within(menu).getByRole("menuitem", { name: /Data Structures/ })).toHaveClass("xl-pathsw__opt--active");
    expect(within(menu).getByText(/System Design Interviews/).closest("[aria-disabled]")).toHaveAttribute("aria-disabled", "true");
    expect(within(menu).getByRole("menuitem", { name: /Browse all paths/ })).toHaveAttribute("href", "/xlearn");
  });

  it("active: a second course renders its own nav (Exercises, no Mock) and calls its own endpoints", async () => {
    const fn = withCatalog(catalog(DSA_PATH, ZZ_FIXTURE_PATH), (url) => {
      if (url.endsWith("/api/paths/zz-fixture/revision/due")) return { status: 200, body: { items: [], dueCount: 3 } };
      return { status: 404 };
    });
    renderAt("/xlearn/zz-fixture/revision");
    const nav = await screen.findByRole("navigation");
    await waitFor(() => expect(within(nav).getByRole("link", { name: /revision/i })).toHaveTextContent("3"));
    const labels = within(nav).getAllByRole("link").map((a) => a.textContent);
    expect(labels).toEqual(["Today", "Roadmap", "Exercises", "Progress", "Revision3", "Mistakes"]);
    expect(within(nav).getByRole("link", { name: "Exercises" })).toHaveAttribute("href", "/xlearn/zz-fixture/problems");
    expect(screen.getByRole("button", { name: /switch curriculum/i }).querySelector(".xl-pathsw__ic")?.textContent).toBe("ZZ");
    await waitFor(() => expect(requested(fn)).toContain("/paths/zz-fixture/mistakes"));
    expect(requested(fn).filter((u) => u.includes("/dsa"))).toEqual([]);
  });

  it("active: the SPA calls the course-scoped endpoints, never the v1 aliases", async () => {
    const fn = withCatalog(catalog());
    renderAt("/xlearn/dsa/dashboard");
    await screen.findByRole("navigation");
    await waitFor(() =>
      expect(requested(fn)).toEqual(
        expect.arrayContaining(["/paths/dsa/dashboard", "/paths/dsa/revision/due", "/paths/dsa/mistakes"]),
      ),
    );
    for (const alias of ["/dashboard", "/progress", "/revision/due", "/mistakes", "/weak-area", "/mocks/trend"]) {
      expect(requested(fn)).not.toContain(alias);
    }
  });

  it("coming_soon: every sub-route renders the AB02-F4 teaser in the plain frame, with no course data calls", async () => {
    for (const path of ["/xlearn/system-design", "/xlearn/system-design/week/3", "/xlearn/system-design/bogus"]) {
      const fn = withCatalog(catalog());
      const { unmount } = renderAt(path);
      const h1 = await screen.findByRole("heading", { level: 1, name: "System Design Interviews" });
      expect(screen.getByText("Learning path · /xlearn/system-design")).toBeInTheDocument();
      expect(screen.getByText(/Scalable systems, trade-offs/)).toBeInTheDocument();
      expect(screen.getByText("12 weeks · 40 problems")).toBeInTheDocument();
      expect(screen.getByText("This course isn’t open yet. It opens here when it’s ready.")).toBeInTheDocument();
      expect(document.querySelector(".xl-content .xl-lock")?.textContent).toContain("Coming soon");
      const back = screen.getByRole("link", { name: /back to catalog/i });
      expect(back).toHaveAttribute("href", "/xlearn");
      // Back to catalog is the first (and only) focusable in the card, after the heading.
      const card = h1.closest(".ds-card")!;
      expect([...card.querySelectorAll("a, button, input, [tabindex]")]).toEqual([back]);
      expect(h1.compareDocumentPosition(back) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
      // Plain frame: brand lockup, no sidebar, no switcher; the coach FAB stays.
      expect(screen.getByRole("link", { name: /all paths/i })).toBeInTheDocument();
      expect(document.querySelector(".xl-side")).toBeNull();
      expect(screen.queryByRole("button", { name: /switch curriculum/i })).not.toBeInTheDocument();
      expect(screen.getByRole("button", { name: /open ai coach/i })).toBeInTheDocument();
      expect(requested(fn).every((u) => ACCOUNT_CALLS.has(u))).toBe(true);
      unmount();
      restoreFetch();
    }
  });

  it("coming_soon: a fixture coming-soon course gets the same teaser", async () => {
    withCatalog(catalog(DSA_PATH, ZZ_SOON_PATH));
    renderAt("/xlearn/zz-soon/dashboard");
    expect(await screen.findByRole("heading", { level: 1, name: "Soon Course" })).toBeInTheDocument();
    expect(screen.getByText("6 weeks · 20 problems")).toBeInTheDocument();
  });

  it("unknown: an unlisted course (unknown, retired, hidden preview) is v1's NotFound in the plain frame", async () => {
    const paths = ["/xlearn/nope", "/xlearn/zz-retired/week/3", "/xlearn/zz-preview", "/xlearn/dsa/bogus", "/xlearn/u"];
    for (const path of paths) {
      const fn = withCatalog(catalog());
      const { unmount } = renderAt(path);
      expect(await screen.findByRole("heading", { level: 1, name: "Page not found" })).toBeInTheDocument();
      expect(screen.getByText("404")).toBeInTheDocument();
      expect(screen.getByText("That route doesn’t exist in xLearn.")).toBeInTheDocument();
      // AB02-F6: v1's target, the default course's dashboard.
      expect(screen.getByRole("link", { name: /back to today/i })).toHaveAttribute("href", "/xlearn/dsa/dashboard");
      expect(document.querySelector(".xl-side")).toBeNull();
      expect(screen.getByRole("link", { name: /all paths/i })).toBeInTheDocument();
      expect(requested(fn).every((u) => ACCOUNT_CALLS.has(u))).toBe(true);
      unmount();
      restoreFetch();
    }
  });

  it("active: a preview course the gateway does list (cohort, m1-04) is treated as active", async () => {
    withCatalog(catalog(DSA_PATH, { ...ZZ_FIXTURE_PATH, status: "preview" }));
    renderAt("/xlearn/zz-fixture");
    expect(await screen.findByRole("navigation")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /switch curriculum/i }));
    expect(within(screen.getByRole("menu")).getByText("Preview")).toBeInTheDocument();
  });

  it("loading: shows the app-level spinner — no frame, no course data calls — until the catalog answers", async () => {
    const fn = installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      // A body that never finishes: the catalog stays pending.
      if (url.endsWith("/api/paths")) return new Response(new ReadableStream({ start() {} }), { status: 200 });
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/dashboard");
    await waitFor(() => expect(requested(fn)).toContain("/me"));
    expect(await screen.findByRole("status")).toHaveTextContent("Loading…");
    // Give any premature fetch a chance to fire.
    await new Promise((r) => setTimeout(r, 50));
    expect(document.querySelector(".xl-app")).toBeNull();
    expect(screen.queryByText("Page not found")).not.toBeInTheDocument();
    expect(requested(fn).every((u) => ACCOUNT_CALLS.has(u))).toBe(true);
  });

  it("error: the catalog failing shows the retry state, not NotFound", async () => {
    let fail = true;
    installFetchMock((url) => {
      if (url.endsWith("/api/me")) return { status: 200, body: authedMe("dsa") };
      if (url.endsWith("/api/paths"))
        return fail ? { status: 500, body: { error: { code: "internal" } } } : { status: 200, body: catalog() };
      return { status: 404 };
    });
    renderAt("/xlearn/dsa/dashboard");
    expect(await screen.findByText("Couldn’t reach xLearn.")).toBeInTheDocument();
    expect(screen.queryByText("Page not found")).not.toBeInTheDocument();
    fail = false;
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(await screen.findByRole("navigation")).toBeInTheDocument();
  });
});
