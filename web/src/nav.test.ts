import { describe, expect, it } from "vitest";
import { buildCrumbs, navForPath } from "./nav";

describe("navForPath (F001: curriculum-scoped nav)", () => {
  const labels = (slug: string) => navForPath(slug).flatMap((s) => s.items.map((i) => i.label));

  it("scopes every nav target under the path slug", () => {
    for (const item of navForPath("dsa").flatMap((s) => s.items)) {
      expect(item.to.startsWith("/dsa")).toBe(true);
    }
  });

  it("keeps the cross-curriculum staples", () => {
    expect(labels("dsa")).toEqual(expect.arrayContaining(["Today", "Roadmap", "Mock interview"]));
  });

  it("drops Settings and Progress (they live in the avatar menu)", () => {
    const l = labels("dsa");
    expect(l).not.toContain("Settings");
    expect(l).not.toContain("Progress");
  });
});

describe("buildCrumbs", () => {
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
});
