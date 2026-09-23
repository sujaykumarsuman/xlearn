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

/** Live counts the Sidebar feeds into the Practice-loop badges. */
export interface NavCounts {
  /** Reviews due today (GET /revision/due → dueCount). */
  revisionDue?: number;
  /** Open journal entries (GET /mistakes → openCount). */
  mistakesOpen?: number;
}

/**
 * navForPath builds the left-nav for a curriculum. The nav is curriculum-SCOPED
 * (F001): it renders only inside a path, and each path can expose its own options —
 * a System Design path need not carry DSA's five-touch "Practice loop". Today,
 * Roadmap and Mock interview are the cross-curriculum staples. Settings + Progress are
 * intentionally absent (they live in the avatar menu). The Practice-loop badges are LIVE
 * counts the Sidebar passes in (reviews due · open mistakes); each is hidden until it
 * loads and only shows when > 0, so a chip means there's something to act on.
 *
 * Only `dsa` is populated today; the switch is the extension point for future paths.
 */
export function navForPath(slug: string, counts: NavCounts = {}): NavSection[] {
  const base = `/${slug}`;
  const badge = (n: number | undefined, teal: boolean): NavItem["badge"] =>
    n && n > 0 ? { text: n > 99 ? "99+" : String(n), teal } : undefined;
  switch (slug) {
    default:
      return [
        {
          items: [
            { label: "Today", icon: "today", to: `${base}/dashboard` },
            { label: "Roadmap", icon: "map", to: base, end: true },
            { label: "Problems", icon: "list", to: `${base}/problems` },
            // Per-course progress is curriculum-scoped now (F009): it moved out of the
            // avatar menu (which now links to the public dashboard) into the course nav.
            { label: "Progress", icon: "chart", to: `${base}/progress` },
          ],
        },
        {
          cap: "Practice loop",
          items: [
            { label: "Revision", icon: "refresh", to: `${base}/revision`, badge: badge(counts.revisionDue, true) },
            { label: "Mistakes", icon: "journal", to: `${base}/mistakes`, badge: badge(counts.mistakesOpen, false) },
            { label: "Mock interview", icon: "target", to: `${base}/mock` },
          ],
        },
      ];
  }
}

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

  // Problem: /dsa/problem/:id — there is no /dsa/problem index, so link the middle crumb
  // to the Problems arena (/dsa/problems), not the (404) singular path.
  if (segs.length === 3 && segs[0] === "dsa" && segs[1] === "problem") {
    crumbs.push({ label: "dsa", to: "/dsa" }, { label: "problems", to: "/dsa/problems" }, { label: `#${segs[2]}` });
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
