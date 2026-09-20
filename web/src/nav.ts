import type { IconName } from "./components/Icon";

/** DOM id linking the sidebar's collapse <label> to the pure-CSS checkbox in
 *  the app frame (the `.xl-collapse-cb:checked ~ .xl-side` toggle in theme.css). */
export const SIDEBAR_COLLAPSE_ID = "xl-collapse";

/** A sidebar navigation entry. */
export interface NavItem {
  label: string;
  icon: IconName;
  to: string;
  /** Exact-match active state (for parent routes like /dsa). */
  end?: boolean;
  badge?: { text: string; teal?: boolean };
}

/** A titled group of nav entries. */
export interface NavSection {
  cap?: string;
  items: NavItem[];
}

// The sidebar structure from the Dashboard/Roadmap artboards. Badge counts are
// static scaffold decoration until the practice/review services back them.
export const NAV: NavSection[] = [
  {
    items: [
      { label: "Today", icon: "today", to: "/dsa/dashboard" },
      { label: "Roadmap", icon: "map", to: "/dsa", end: true },
    ],
  },
  {
    cap: "Practice loop",
    items: [
      { label: "Revision", icon: "refresh", to: "/dsa/revision", badge: { text: "4", teal: true } },
      { label: "Mistakes", icon: "journal", to: "/dsa/mistakes", badge: { text: "3" } },
      { label: "Mock interview", icon: "target", to: "/dsa/mock" },
      { label: "Progress", icon: "chart", to: "/dsa/progress" },
    ],
  },
  {
    cap: "Account",
    items: [{ label: "Settings", icon: "settings", to: "/settings" }],
  },
];

/** One breadcrumb: a label, and a link target unless it is the current page. */
export interface Crumb {
  label: string;
  to?: string;
}

/**
 * buildCrumbs turns an app-relative path (+ query string) into the top-bar
 * breadcrumb trail. Two content routes get bespoke trails so the last crumb reads
 * like the artboards and Concept shows the week it was opened from:
 *   - /dsa/week/:n           → xlearn / dsa / week/N
 *   - /dsa/concept/:slug     → xlearn / dsa / [week N /] concept/slug   (week from ?week=N)
 * Every other route falls back to one crumb per path segment. The leading `xlearn`
 * links home except when it is the only (current-page) crumb.
 */
export function buildCrumbs(pathname: string, search = ""): Crumb[] {
  const segs = pathname.replace(/\/+$/, "").split("/").filter(Boolean);
  const crumbs: Crumb[] = [{ label: "xlearn", to: "/" }];

  if (segs.length === 3 && segs[0] === "dsa" && segs[1] === "week") {
    crumbs.push({ label: "dsa", to: "/dsa" }, { label: `week/${segs[2]}` });
    return crumbs;
  }

  if (segs.length === 3 && segs[0] === "dsa" && segs[1] === "concept") {
    crumbs.push({ label: "dsa", to: "/dsa" });
    const week = new URLSearchParams(search).get("week");
    if (week && /^\d+$/.test(week)) {
      crumbs.push({ label: `week ${week}`, to: `/dsa/week/${week}` });
    }
    crumbs.push({ label: `concept/${segs[2]}` });
    return crumbs;
  }

  segs.forEach((seg, i) => {
    const last = i === segs.length - 1;
    crumbs.push(last ? { label: seg } : { label: seg, to: "/" + segs.slice(0, i + 1).join("/") });
  });
  return crumbs;
}

/**
 * routeTitle returns a human screen name for an app-relative path (the value of
 * useLocation().pathname, which excludes the router basename). Used for the
 * coach context chip.
 */
export function routeTitle(pathname: string): string {
  const p = pathname.replace(/\/+$/, "") || "/";
  if (p === "/") return "Catalog";
  if (p === "/dsa") return "Roadmap";
  if (p === "/dsa/dashboard") return "Today";
  if (p.startsWith("/dsa/week/")) return "Week";
  if (p.startsWith("/dsa/concept/")) return "Concept";
  if (p.startsWith("/dsa/problem/")) return "Problem";
  if (p === "/dsa/revision") return "Revision";
  if (p === "/dsa/mistakes") return "Mistakes";
  if (p === "/dsa/mock") return "Mock interview";
  if (p === "/dsa/progress") return "Progress";
  if (p === "/settings") return "Settings";
  if (p === "/auth") return "Sign in";
  return "xLearn";
}
