import type { IconName } from "./components/Icon";
import { coursePath } from "./lib/course";
import { isReservedSegment } from "./lib/courseSlugGuard";
import type { CourseViewNav, NavScreen } from "./lib/curriculum";

/** DOM id linking the sidebar's collapse <label> to the pure-CSS checkbox in
 *  the app frame (the `.xl-collapse-cb:checked ~ .xl-side` toggle in theme.css). */
export const SIDEBAR_COLLAPSE_ID = "xl-collapse";

/** A sidebar navigation entry. */
export interface NavItem {
  label: string;
  icon: IconName;
  to: string;
  /** Exact-match active state (for the course root, the Roadmap). */
  end?: boolean;
  badge?: { text: string; teal?: boolean };
}

/** A titled group of nav entries. */
export interface NavSection {
  cap?: string;
  items: NavItem[];
}

/** Live counts the Sidebar feeds into the Practice-loop badges. */
export interface NavCounts {
  /** Reviews due today (GET /paths/{slug}/revision/due → dueCount). */
  revisionDue?: number;
  /** Open journal entries (GET /paths/{slug}/mistakes → openCount). */
  mistakesOpen?: number;
}

/**
 * How each screen of the closed nav set renders: its icon, its URL segment under the
 * course (the same for every course, t0 §7) and, for the Practice-loop screens, which
 * live count drives its badge. The manifest supplies only the labels, the order, the
 * group caps and which screens exist.
 */
const SCREENS: Record<NavScreen, { icon: IconName; segment?: string; badge?: "revision" | "mistakes" }> = {
  today: { icon: "today", segment: "dashboard" },
  // The course root: exact-match active, so it isn't lit on every course page.
  roadmap: { icon: "map" },
  problems: { icon: "list", segment: "problems" },
  // Per-course progress is curriculum-scoped (F009): it lives in the course nav, not the
  // avatar menu (which links to the public dashboard).
  progress: { icon: "chart", segment: "progress" },
  revision: { icon: "refresh", segment: "revision", badge: "revision" },
  mistakes: { icon: "journal", segment: "mistakes", badge: "mistakes" },
  mock: { icon: "target", segment: "mock" },
};

/**
 * STANDARD_NAV is the nav a course view without a nav block gets. Only a pre-v1.7
 * curriculum answers without one (the minutes of a rolling update); every active course
 * manifest declares its own. It is the v1 nav: the standard screens and labels.
 */
const STANDARD_NAV: CourseViewNav = {
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
};

/**
 * navForPath renders a course's left nav from its manifest nav block (the learner-safe
 * course view, ADR-0026 §5). The nav is curriculum-SCOPED (F001): it renders only inside
 * a course, and each course declares its own groups, labels and screens — a course with
 * no mock has no Mock row (AB02-F8). Settings is intentionally absent (it lives in the
 * avatar menu). The Practice-loop badges are LIVE counts the Sidebar passes in (reviews
 * due · open mistakes); each is hidden until it loads and only shows when > 0, so a chip
 * means there's something to act on. A screen outside the closed set is skipped.
 */
export function navForPath(slug: string, nav: CourseViewNav | undefined, counts: NavCounts = {}): NavSection[] {
  const badge = (n: number | undefined, teal: boolean): NavItem["badge"] =>
    n && n > 0 ? { text: n > 99 ? "99+" : String(n), teal } : undefined;
  const sections: NavSection[] = [];
  for (const group of (nav ?? STANDARD_NAV).groups) {
    const items: NavItem[] = [];
    for (const { screen, label } of group.items) {
      if (!Object.hasOwn(SCREENS, screen)) continue;
      const def = SCREENS[screen];
      const to = def.segment ? coursePath(slug, def.segment) : coursePath(slug);
      if (def.badge === "revision") items.push({ label, icon: def.icon, to, badge: badge(counts.revisionDue, true) });
      else if (def.badge === "mistakes") items.push({ label, icon: def.icon, to, badge: badge(counts.mistakesOpen, false) });
      else if (!def.segment) items.push({ label, icon: def.icon, to, end: true });
      else items.push({ label, icon: def.icon, to });
    }
    if (items.length === 0) continue;
    sections.push(group.cap ? { cap: group.cap, items } : { items });
  }
  return sections;
}

/** navHasScreen reports whether a course's nav shows a screen (so the Sidebar only fetches
 *  a badge count the nav will render). */
export function navHasScreen(nav: CourseViewNav | undefined, screen: NavScreen): boolean {
  return (nav ?? STANDARD_NAV).groups.some((g) => g.items.some((it) => it.screen === screen));
}

/** One breadcrumb: a label, and a link target unless it is the current page. */
export interface Crumb {
  label: string;
  to?: string;
}

/** courseOf returns the course segment of a path's segments, or null when the first
 *  segment is absent or reserved (/settings, /auth, /u/…): those aren't course routes. */
function courseOf(segs: string[]): string | null {
  const first = segs[0];
  return first && !isReservedSegment(first) ? first : null;
}

/**
 * buildCrumbs turns an app-relative path (+ query string) into the top-bar breadcrumb
 * trail (AB02-F7: v1's rules, now for any course). Three course routes get bespoke
 * trails so the last crumb reads like the artboards and Concept shows the week it was
 * opened from:
 *   - /:course/week/:n         → xlearn / <course> / week/N
 *   - /:course/concept/:slug   → xlearn / <course> / [week N /] concept/slug   (week from ?week=N)
 *   - /:course/problem/:id     → xlearn / <course> / problems / #id
 * Every other route falls back to one crumb per path segment. The leading `xlearn`
 * links home except when it is the only (current-page) crumb. Crumbs show the slug, not
 * the title (v1).
 */
export function buildCrumbs(pathname: string, search = ""): Crumb[] {
  const segs = pathname.replace(/\/+$/, "").split("/").filter(Boolean);
  const crumbs: Crumb[] = [{ label: "xlearn", to: "/" }];
  const course = courseOf(segs);

  if (course && segs.length === 3 && segs[1] === "week") {
    crumbs.push({ label: course, to: `/${course}` }, { label: `week/${segs[2]}` });
    return crumbs;
  }

  if (course && segs.length === 3 && segs[1] === "concept") {
    crumbs.push({ label: course, to: `/${course}` });
    const week = new URLSearchParams(search).get("week");
    if (week && /^\d+$/.test(week)) {
      crumbs.push({ label: `week ${week}`, to: `/${course}/week/${week}` });
    }
    crumbs.push({ label: `concept/${segs[2]}` });
    return crumbs;
  }

  // Problem: /:course/problem/:id — there is no singular problem index, so link the
  // middle crumb to the Problems arena (/:course/problems), not the (404) singular path.
  if (course && segs.length === 3 && segs[1] === "problem") {
    crumbs.push(
      { label: course, to: `/${course}` },
      { label: "problems", to: `/${course}/problems` },
      { label: `#${segs[2]}` },
    );
    return crumbs;
  }

  segs.forEach((seg, i) => {
    const last = i === segs.length - 1;
    crumbs.push(last ? { label: seg } : { label: seg, to: "/" + segs.slice(0, i + 1).join("/") });
  });
  return crumbs;
}

/** The screen name of each single-segment course route (/:course/<segment>). */
const COURSE_SCREEN_TITLE = new Map([
  ["dashboard", "Today"],
  ["revision", "Revision"],
  ["mistakes", "Mistakes"],
  ["mock", "Mock interview"],
  ["progress", "Progress"],
]);

/** The screen name of each item route (/:course/<segment>/…). */
const COURSE_ITEM_TITLE = new Map([
  ["week", "Week"],
  ["concept", "Concept"],
  ["problem", "Problem"],
]);

/**
 * routeTitle returns a human screen name for an app-relative path (the value of
 * useLocation().pathname, which excludes the router basename), generic over
 * /:course/*. Used for the coach context chip.
 */
export function routeTitle(pathname: string): string {
  const p = pathname.replace(/\/+$/, "") || "/";
  if (p === "/") return "Catalog";
  if (p === "/settings") return "Settings";
  if (p === "/auth") return "Sign in";
  const segs = p.split("/").filter(Boolean);
  const [, sub] = segs;
  if (courseOf(segs)) {
    if (sub === undefined) return "Roadmap";
    const title = segs.length === 2 ? COURSE_SCREEN_TITLE.get(sub) : COURSE_ITEM_TITLE.get(sub);
    if (title) return title;
  }
  return "xLearn";
}
