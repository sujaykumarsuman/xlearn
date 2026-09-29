import { describe, expect, it } from "vitest";
import { buildCrumbs, navForPath, routeTitle, type NavSection } from "./nav";
import { DSA_VIEW, ZZ_FIXTURE_VIEW } from "./test/courses";

/** V1_DSA_NAV is v1.6.0's navForPath("dsa") output, written out (web/src/nav.ts before
 *  m1-03): the manifest-driven nav must reproduce it exactly for the DSA course. */
function v1DsaNav(revisionDue?: number, mistakesOpen?: number): NavSection[] {
  const badge = (n: number | undefined, teal: boolean) =>
    n && n > 0 ? { text: n > 99 ? "99+" : String(n), teal } : undefined;
  return [
    {
      items: [
        { label: "Today", icon: "today", to: "/dsa/dashboard" },
        { label: "Roadmap", icon: "map", to: "/dsa", end: true },
        { label: "Problems", icon: "list", to: "/dsa/problems" },
        { label: "Progress", icon: "chart", to: "/dsa/progress" },
      ],
    },
    {
      cap: "Practice loop",
      items: [
        { label: "Revision", icon: "refresh", to: "/dsa/revision", badge: badge(revisionDue, true) },
        { label: "Mistakes", icon: "journal", to: "/dsa/mistakes", badge: badge(mistakesOpen, false) },
        { label: "Mock interview", icon: "target", to: "/dsa/mock" },
      ],
    },
  ];
}

describe("navForPath (F001 + AB02: the manifest nav block)", () => {
  it("renders the DSA view identically to v1 (no counts)", () => {
    expect(navForPath("dsa", DSA_VIEW.nav)).toStrictEqual(v1DsaNav());
  });

  it("renders the DSA view identically to v1 (live badge counts, incl. the 99+ cap and hidden zeros)", () => {
    expect(navForPath("dsa", DSA_VIEW.nav, { revisionDue: 4, mistakesOpen: 3 })).toStrictEqual(v1DsaNav(4, 3));
    expect(navForPath("dsa", DSA_VIEW.nav, { revisionDue: 120, mistakesOpen: 0 })).toStrictEqual(v1DsaNav(120, 0));
    expect(navForPath("dsa", DSA_VIEW.nav, { revisionDue: 120 })[1]!.items[0]!.badge).toEqual({ text: "99+", teal: true });
  });

  it("falls back to the v1 nav when the view has no nav block (a pre-v1.7 curriculum)", () => {
    expect(navForPath("dsa", undefined, { revisionDue: 2 })).toStrictEqual(v1DsaNav(2));
  });

  it("renders a second course from its own manifest: its labels, same URL segments, no Mock row (AB02-F8)", () => {
    const nav = navForPath("zz-fixture", ZZ_FIXTURE_VIEW.nav, { revisionDue: 2 });
    expect(nav).toStrictEqual([
      {
        items: [
          { label: "Today", icon: "today", to: "/zz-fixture/dashboard" },
          { label: "Roadmap", icon: "map", to: "/zz-fixture", end: true },
          { label: "Exercises", icon: "list", to: "/zz-fixture/problems" },
          { label: "Progress", icon: "chart", to: "/zz-fixture/progress" },
        ],
      },
      {
        cap: "Practice loop",
        items: [
          { label: "Revision", icon: "refresh", to: "/zz-fixture/revision", badge: { text: "2", teal: true } },
          { label: "Mistakes", icon: "journal", to: "/zz-fixture/mistakes", badge: undefined },
        ],
      },
    ]);
    const labels = nav.flatMap((s) => s.items.map((i) => i.label));
    expect(labels).not.toContain("Mock interview");
    expect(labels).not.toContain("Problems");
  });

  it("keeps Settings out of the nav, but surfaces per-course Progress (F009)", () => {
    const l = navForPath("dsa", DSA_VIEW.nav).flatMap((s) => s.items.map((i) => i.label));
    expect(l).not.toContain("Settings");
    expect(l).toContain("Progress");
  });

  it("skips a screen outside the closed set, and a group left empty", () => {
    const nav = navForPath("zz-fixture", {
      item_noun: "x",
      groups: [
        { cap: "Odd", items: [{ screen: "constructor" as never, label: "Nope" }] },
        { items: [{ screen: "mock", label: "Mock" }] },
      ],
    });
    expect(nav).toStrictEqual([{ items: [{ label: "Mock", icon: "target", to: "/zz-fixture/mock" }] }]);
  });
});

describe("buildCrumbs (AB02-F7: v1 rules, generic over /:course)", () => {
  it("marks the home crumb as the current page at the root", () => {
    expect(buildCrumbs("/")).toEqual([{ label: "xlearn", to: "/" }]);
  });

  it("links each ancestor segment on a plain route", () => {
    expect(buildCrumbs("/settings")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "settings" },
    ]);
  });

  it("collapses the Week route so the last crumb reads week/N", () => {
    expect(buildCrumbs("/dsa/week/2")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "dsa", to: "/dsa" },
      { label: "week/2" },
    ]);
  });

  it("adds the parent week crumb for a Concept opened from a week", () => {
    expect(buildCrumbs("/dsa/concept/sliding-window", "?week=2")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "dsa", to: "/dsa" },
      { label: "week 2", to: "/dsa/week/2" },
      { label: "concept/sliding-window" },
    ]);
  });

  it("links the Problem crumb to the Problems arena (not the 404 singular path)", () => {
    expect(buildCrumbs("/dsa/problem/16")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "dsa", to: "/dsa" },
      { label: "problems", to: "/dsa/problems" },
      { label: "#16" },
    ]);
  });

  it("omits the week crumb for a Concept reached directly", () => {
    expect(buildCrumbs("/dsa/concept/sliding-window")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "dsa", to: "/dsa" },
      { label: "concept/sliding-window" },
    ]);
  });

  it("ignores a non-numeric week query param", () => {
    expect(buildCrumbs("/dsa/concept/sliding-window", "?week=abc")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "dsa", to: "/dsa" },
      { label: "concept/sliding-window" },
    ]);
  });

  it("falls back to one crumb per segment on the other course routes (v1: xlearn / dsa / revision)", () => {
    expect(buildCrumbs("/dsa/revision")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "dsa", to: "/dsa" },
      { label: "revision" },
    ]);
    expect(buildCrumbs("/dsa")).toEqual([{ label: "xlearn", to: "/" }, { label: "dsa" }]);
  });

  it("applies the same rules to any course (AB02-F7 last row)", () => {
    expect(buildCrumbs("/go-concurrency/week/2")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "go-concurrency", to: "/go-concurrency" },
      { label: "week/2" },
    ]);
    expect(buildCrumbs("/zz-fixture/concept/fan-out", "?week=3")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "zz-fixture", to: "/zz-fixture" },
      { label: "week 3", to: "/zz-fixture/week/3" },
      { label: "concept/fan-out" },
    ]);
    expect(buildCrumbs("/zz-fixture/problem/zz-7")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "zz-fixture", to: "/zz-fixture" },
      { label: "problems", to: "/zz-fixture/problems" },
      { label: "#zz-7" },
    ]);
  });

  it("never treats a reserved first segment as a course", () => {
    expect(buildCrumbs("/u/ada/week")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "u", to: "/u" },
      { label: "ada", to: "/u/ada" },
      { label: "week" },
    ]);
    expect(buildCrumbs("/settings/week/3")).toEqual([
      { label: "xlearn", to: "/" },
      { label: "settings", to: "/settings" },
      { label: "week", to: "/settings/week" },
      { label: "3" },
    ]);
  });
});

describe("routeTitle (generic over /:course)", () => {
  it.each([
    ["/", "Catalog"],
    ["/settings", "Settings"],
    ["/auth", "Sign in"],
    ["/dsa", "Roadmap"],
    ["/dsa/", "Roadmap"],
    ["/dsa/dashboard", "Today"],
    ["/dsa/week/3", "Week"],
    ["/dsa/concept/two-pointers", "Concept"],
    ["/dsa/problem/16", "Problem"],
    ["/dsa/revision", "Revision"],
    ["/dsa/mistakes", "Mistakes"],
    ["/dsa/mock", "Mock interview"],
    ["/dsa/progress", "Progress"],
    // v1 gave the arena and unknown sub-routes no screen name.
    ["/dsa/problems", "xLearn"],
    ["/dsa/week", "xLearn"],
    ["/dsa/constructor", "xLearn"],
    ["/zz-fixture/week/2", "Week"],
    ["/zz-fixture", "Roadmap"],
    ["/u/ada", "xLearn"],
  ])("%s → %s", (path, want) => {
    expect(routeTitle(path)).toBe(want);
  });
});
