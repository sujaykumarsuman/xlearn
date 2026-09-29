import { describe, expect, it } from "vitest";
import { DSA_PATH, ZZ_FIXTURE_PATH } from "../test/courses";
import { courseShortCode, coursePath, firstSegment, isCourseCandidate, languageNote } from "./course";
import { DEFAULT_COURSE } from "./defaultCourse";

describe("coursePath", () => {
  it("builds the v1 links for the DSA course", () => {
    expect(coursePath("dsa")).toBe("/dsa");
    expect(coursePath("dsa", "dashboard")).toBe("/dsa/dashboard");
    expect(coursePath("dsa", "week", 3)).toBe("/dsa/week/3");
    expect(coursePath("dsa", "problem", "16")).toBe("/dsa/problem/16");
    expect(`${coursePath("dsa", "concept", "two-pointers")}?week=2`).toBe("/dsa/concept/two-pointers?week=2");
  });

  it("keeps every part one path segment", () => {
    expect(coursePath("zz-fixture", "problem", "a/b c")).toBe("/zz-fixture/problem/a%2Fb%20c");
  });
});

describe("course helpers", () => {
  it("reads the first path segment", () => {
    expect(firstSegment("/")).toBe("");
    expect(firstSegment("/dsa/week/3")).toBe("dsa");
    expect(firstSegment("/zz-fixture")).toBe("zz-fixture");
  });

  it("never treats a reserved or empty segment as a course", () => {
    expect(isCourseCandidate("dsa")).toBe(true);
    expect(isCourseCandidate("nope")).toBe(true);
    for (const seg of ["", "settings", "auth", "u", "api", "assets", "healthz", "readyz", "privacy"]) {
      expect(isCourseCandidate(seg), seg).toBe(false);
    }
  });

  it("derives the short code and language note from the course view", () => {
    expect(courseShortCode(DSA_PATH)).toBe("DSA");
    expect(courseShortCode(ZZ_FIXTURE_PATH)).toBe("ZZ");
    // The v1 fallback: the slug head.
    expect(courseShortCode({ slug: "go-concurrency" })).toBe("GOC");
    expect(languageNote(DSA_PATH.course)).toBe("Go-first");
    expect(languageNote(undefined)).toBe("");
    expect(languageNote({ ...ZZ_FIXTURE_PATH.course!, primary_language: "python" })).toBe("Python-first");
  });

  it("has exactly one SPA default course: course.DefaultSlug's (AB02-F6 NotFound target)", () => {
    expect(DEFAULT_COURSE).toBe("dsa");
  });
});
