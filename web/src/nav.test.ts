import { describe, expect, it } from "vitest";
import { buildCrumbs } from "./nav";

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
