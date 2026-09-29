// Course fixtures for the SPA tests: the learner-safe course views (internal/course/view.go
// `LearnerView()`) of the real DSA manifest (curriculum/courses/dsa/course.json) and of the
// Go test fixtures (internal/course/testdata/fixtures: zz-fixture active with "Exercises"
// and no mock, zz-soon coming_soon), plus GET /paths catalog bodies built from them.
import type { CourseView, Path, PathsResponse } from "../lib/curriculum";

/** The DSA course view, as curriculum serves it. */
export const DSA_VIEW: CourseView = {
  slug: "dsa",
  title: "Data Structures & Algorithms",
  status: "active",
  short_code: "DSA",
  nav: {
    item_noun: "problem",
    groups: [
      {
        items: [
          { screen: "today", label: "Today" },
          { screen: "roadmap", label: "Roadmap" },
          { screen: "problems", label: "Problems" },
          { screen: "progress", label: "Progress" },
        ],
      },
      {
        cap: "Practice loop",
        items: [
          { screen: "revision", label: "Revision" },
          { screen: "mistakes", label: "Mistakes" },
          { screen: "mock", label: "Mock interview" },
        ],
      },
    ],
  },
  stages: {
    attempt: { duration_s: 900, label: "Attempt" },
    hint: { duration_s: 600, label: "Hint" },
    solution: { label: "Solution" },
    reimplement: "required",
  },
  est_minutes: { course_attempt: 45, touch_resolve: 20, touch_recall: 5, mock: 45 },
  mock_rail: [
    { label: "Clarify", start_min: 0, end_min: 5 },
    { label: "Brute force", start_min: 5, end_min: 10 },
    { label: "Observation → plan", start_min: 10, end_min: 18 },
    { label: "Code", start_min: 18, end_min: 33 },
    { label: "Trace + edges", start_min: 33, end_min: 40 },
    { label: "Complexity + follow-ups", start_min: 40, end_min: 45 },
  ],
  primary_language: "go",
};

/** The zz-fixture view: active, its own item label ("Exercises"), no mock. */
export const ZZ_FIXTURE_VIEW: CourseView = {
  slug: "zz-fixture",
  title: "Fixture Course",
  status: "active",
  short_code: "ZZ",
  nav: {
    item_noun: "exercise",
    groups: [
      {
        items: [
          { screen: "today", label: "Today" },
          { screen: "roadmap", label: "Roadmap" },
          { screen: "problems", label: "Exercises" },
          { screen: "progress", label: "Progress" },
        ],
      },
      {
        cap: "Practice loop",
        items: [
          { screen: "revision", label: "Revision" },
          { screen: "mistakes", label: "Mistakes" },
        ],
      },
    ],
  },
  primary_language: "go",
};

/** The DSA catalog entry. */
export const DSA_PATH: Path = {
  slug: "dsa",
  title: "Data Structures & Algorithms",
  status: "active",
  summary: "From arrays to graphs and DP, built to make you interview-ready — and to keep it retained.",
  problem_total: 151,
  week_total: 16,
  course: DSA_VIEW,
};

/** The zz-fixture catalog entry. */
export const ZZ_FIXTURE_PATH: Path = {
  slug: "zz-fixture",
  title: "Fixture Course",
  status: "active",
  summary: "A test-only course.",
  problem_total: 12,
  week_total: 4,
  course: ZZ_FIXTURE_VIEW,
};

/** A coming_soon catalog entry (no nav in its view). */
export function comingSoonPath(slug: string, title: string, shortCode: string, weeks: number, problems: number): Path {
  return {
    slug,
    title,
    status: "coming_soon",
    summary: `${title}: coming soon.`,
    problem_total: problems,
    week_total: weeks,
    course: { slug, title, status: "coming_soon", short_code: shortCode },
  };
}

/** The system-design teaser entry (AB02-F4's sample). */
export const SYSTEM_DESIGN_PATH: Path = {
  ...comingSoonPath("system-design", "System Design Interviews", "SD", 12, 40),
  summary: "Scalable systems, trade-offs, and the whiteboard narrative — timed design rounds with rubric scoring.",
};

/** The zz-soon fixture entry. */
export const ZZ_SOON_PATH: Path = comingSoonPath("zz-soon", "Soon Course", "ZZS", 6, 20);

/** catalog builds a GET /paths body. The default is the production shape: the DSA course
 *  active, the other courses coming soon. */
export function catalog(...paths: Path[]): PathsResponse {
  return {
    paths: paths.length
      ? paths
      : [
          DSA_PATH,
          SYSTEM_DESIGN_PATH,
          comingSoonPath("go-concurrency", "Go Concurrency Patterns", "GC", 8, 60),
          comingSoonPath("lld-ood", "Low-Level / OOD", "LLD", 6, 24),
          comingSoonPath("sql", "SQL & Data Modeling", "SQL", 6, 50),
          comingSoonPath("behavioral", "Behavioral & Storytelling", "BEH", 4, 20),
        ],
  };
}
