import { describe, expect, it } from "vitest";
import { routes } from "../router";
import { RESERVED_SEGMENTS, isReservedSegment } from "./courseSlugGuard";
import reserved from "./reservedSegments.json";

describe("course-slug guard (SPA side)", () => {
  // The Go test (internal/course, TestReservedSegmentsJSON) pins reservedSegments.json to
  // the Go guard; this pins the SPA module to the JSON, so the three agree.
  it("exports exactly the Go guard's list (reservedSegments.json)", () => {
    expect([...RESERVED_SEGMENTS]).toStrictEqual(reserved.reserved);
  });

  it("reserves the static SPA segments and the gateway-owned paths (AB02 routes frame)", () => {
    expect([...RESERVED_SEGMENTS].sort()).toStrictEqual(
      ["api", "assets", "auth", "healthz", "privacy", "readyz", "settings", "u"].sort(),
    );
    for (const seg of RESERVED_SEGMENTS) expect(isReservedSegment(seg)).toBe(true);
    expect(isReservedSegment("dsa")).toBe(false);
    expect(isReservedSegment("zz-fixture")).toBe(false);
    expect(isReservedSegment("")).toBe(false);
    expect(isReservedSegment("constructor")).toBe(false);
  });

  it("covers every static first segment the router declares", () => {
    // A new static top-level route must join the Go list first, or a course slug could
    // shadow it (the :course segment is dynamic).
    const firstSegments = new Set<string>();
    const walk = (rs: typeof routes, prefix: string) => {
      for (const r of rs) {
        const full = r.path === undefined ? prefix : `${prefix}/${r.path}`.replace(/\/+/g, "/");
        const first = full.split("/").filter(Boolean)[0];
        if (first && !first.startsWith(":") && first !== "*") firstSegments.add(first);
        if (r.children) walk(r.children, full);
      }
    };
    walk(routes, "");
    expect(firstSegments.size).toBeGreaterThan(0);
    for (const seg of firstSegments) expect(isReservedSegment(seg), seg).toBe(true);
  });
});
