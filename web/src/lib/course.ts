// Course resolution for the SPA (sprint m1-03; ADR-0026 §5, t0 §7). Every course lives
// at /xlearn/:course/…, one dynamic route inside AuthedShell, and the URL's first
// segment is resolved against the session-gated catalog (GET /paths). Nothing in the SPA
// names a course: screens read the current one from useCourse() and build every in-app
// course link with coursePath().
import { useLocation } from "react-router-dom";
import { isReservedSegment } from "./courseSlugGuard";
import { usePaths, type CourseStatus, type CourseView, type Path } from "./curriculum";

/**
 * The resolution state of the URL's course:
 *  - `loading`     — the catalog hasn't answered yet (render no course state or data);
 *  - `error`       — the catalog request failed (nothing to resolve against; retry);
 *  - `unknown`     — not a course: a reserved or empty first segment, or a slug the
 *                    catalog doesn't list (unknown, retired, or a preview outside the
 *                    cohort all look the same) → NotFound (AB02-F6);
 *  - `coming_soon` — listed but not open → the coming-soon teaser (AB02-F4);
 *  - `active`      — open (a `preview` course the gateway lists is treated the same).
 */
export type CourseState = "loading" | "error" | "unknown" | "coming_soon" | "active";

/** The current course, as resolved by useCourse(). */
export interface Course {
  /** The URL's first segment ("" at the root): the course slug when state is active. */
  slug: string;
  /** The catalog entry (title, summary, totals), once resolved. */
  path?: Path;
  /** The learner-safe course view (nav, short code, …), once resolved. */
  view?: CourseView;
  /** The catalog status, once resolved. */
  status?: CourseStatus;
  state: CourseState;
  /** Retries the catalog request (for the error state). */
  retry: () => void;
}

/** firstSegment returns the first segment of an app-relative pathname ("" at the root). */
export function firstSegment(pathname: string): string {
  return pathname.split("/").filter(Boolean)[0] ?? "";
}

/** isCourseCandidate reports whether a first segment could name a course: non-empty and
 *  not reserved for a static route or the gateway (courseSlugGuard). */
export function isCourseCandidate(seg: string): boolean {
  return seg !== "" && !isReservedSegment(seg);
}

/**
 * useCourse resolves the course named by the URL's FIRST path segment against the
 * session-gated catalog. It reads the pathname rather than route params so it works in
 * the layout components above the routed screen (CurriculumShell, Sidebar, Topbar, the
 * coach panel) as well as in the screens. The catalog query is shared (["paths"]), so
 * the screens re-use the shell's fetch; a route that can't be a course (the root,
 * /settings, …) resolves to `unknown` without fetching.
 */
export function useCourse(): Course {
  const { pathname } = useLocation();
  const slug = firstSegment(pathname);
  const candidate = isCourseCandidate(slug);
  const paths = usePaths(candidate);
  const retry = () => void paths.refetch();

  if (!candidate) return { slug, state: "unknown", retry };
  // No catalog yet: loading, or failed (a refetch after a failure, e.g. the 401 from
  // before sign-in, counts as loading, not as a failure to show).
  if (!paths.data) return { slug, state: paths.isError && !paths.isFetching ? "error" : "loading", retry };

  const path = paths.data.paths.find((p) => p.slug === slug);
  if (!path) return { slug, state: "unknown", retry };
  const resolved = { slug, path, view: path.course, status: path.status, retry };
  switch (path.status) {
    case "active":
    case "preview":
      return { ...resolved, state: "active" };
    case "coming_soon":
      return { ...resolved, state: "coming_soon" };
    default:
      // A status this SPA doesn't know (retired never reaches it): not a course to show.
      return { slug, state: "unknown", retry };
  }
}

/** COURSE_NOT_FOUND_HANDLE is the route handle of the course subtree's catch-all
 *  (router.tsx): CurriculumShell renders an unknown sub-route of a course (/<course>/bogus)
 *  as NotFound in the PLAIN frame, exactly as v1 did. */
export const COURSE_NOT_FOUND_HANDLE = { courseNotFound: true } as const;

/** isCourseNotFoundHandle reports whether a matched route's handle is the catch-all's. */
export function isCourseNotFoundHandle(handle: unknown): boolean {
  return (handle as Partial<typeof COURSE_NOT_FOUND_HANDLE> | undefined)?.courseNotFound === true;
}

/** useCourseSlug is the current course slug alone (the URL's first segment), for the
 *  sub-components of a course screen that only build links. */
export function useCourseSlug(): string {
  return firstSegment(useLocation().pathname);
}

/**
 * coursePath builds an app-relative link into a course: coursePath(slug) is the Roadmap
 * (`/<slug>`), coursePath(slug, "week", 3) is `/<slug>/week/3`. Each part is
 * URL-encoded, so a problem id or concept slug is always one path segment. Add a query
 * string after it (`${coursePath(slug, "concept", c)}?week=2`).
 */
export function coursePath(slug: string, ...segments: Array<string | number>): string {
  return "/" + [slug, ...segments].map((s) => encodeURIComponent(String(s))).join("/");
}

/** courseShortCode is a course's short mono code (DSA, …): the manifest's short code from
 *  the course view, falling back to the slug head when the view is absent. */
export function courseShortCode(path: Pick<Path, "slug" | "course">): string {
  return path.course?.short_code || path.slug.replace(/[^a-z0-9]/gi, "").slice(0, 3).toUpperCase();
}

// Display names for the manifest's primary-language codes; any other code is shown
// capitalized.
const LANGUAGE_LABEL: Record<string, string> = {
  go: "Go",
  cpp: "C++",
  javascript: "JavaScript",
  typescript: "TypeScript",
  sql: "SQL",
};

/** languageLabel renders a primary-language code for display ("go" → "Go"). */
export function languageLabel(code: string): string {
  return Object.hasOwn(LANGUAGE_LABEL, code)
    ? LANGUAGE_LABEL[code]!
    : code.charAt(0).toUpperCase() + code.slice(1);
}

/** languageNote is a course's "Go-first" note from its view, or "" without a primary
 *  language. */
export function languageNote(view: CourseView | undefined): string {
  return view?.primary_language ? `${languageLabel(view.primary_language)}-first` : "";
}
