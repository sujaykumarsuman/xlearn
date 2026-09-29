import { Fragment } from "react";
import { Link, NavLink } from "react-router-dom";
import type { CourseViewNav } from "../lib/curriculum";
import { useMistakes } from "../lib/mistakes";
import { useDueRevision } from "../lib/revision";
import { navForPath, navHasScreen, SIDEBAR_COLLAPSE_ID } from "../nav";
import { Icon } from "./Icon";

/**
 * The collapsible left sidebar (F001): brand, the course-scoped nav rendered from the
 * course manifest's nav block (AB02), and the pure-CSS collapse toggle. It renders only
 * inside an active course; the course switcher is in the top bar, so the sidebar is
 * purely this course's navigation.
 */
export function Sidebar({ slug, nav: courseNav }: { slug: string; nav?: CourseViewNav }) {
  // Live Practice-loop badges: reviews due today + open journal entries. Both are cached
  // react-query reads (shared with the Revision / Mistakes screens), so this adds no fetch
  // once those screens have loaded; the badge stays hidden until the count resolves. A
  // course whose nav has no Revision / Mistakes row doesn't fetch that count.
  const revision = useDueRevision(slug, navHasScreen(courseNav, "revision"));
  const mistakes = useMistakes(slug, navHasScreen(courseNav, "mistakes"));
  const nav = navForPath(slug, courseNav, {
    revisionDue: revision.data?.dueCount,
    mistakesOpen: mistakes.data?.openCount,
  });

  return (
    <aside className="xl-side">
      <Link to="/" className="xl-brand" aria-label="xLearn — home">
        <div className="xl-brand__mark">x</div>
        <div className="xl-brand__name">
          x<b>Learn</b>
        </div>
      </Link>

      <nav className="xl-nav">
        {nav.map((section, i) => (
          <Fragment key={section.cap ?? `sect-${i}`}>
            {section.cap && <div className="xl-nav__cap">{section.cap}</div>}
            {section.items.map((item) => (
              <NavLink
                key={item.to}
                to={item.to}
                end={item.end}
                className={({ isActive }) =>
                  isActive ? "xl-nav__item xl-nav__item--active" : "xl-nav__item"
                }
              >
                <Icon name={item.icon} />
                {item.label}
                {item.badge && (
                  <span
                    className={
                      item.badge.teal ? "xl-nav__badge xl-nav__badge--teal" : "xl-nav__badge"
                    }
                  >
                    {item.badge.text}
                  </span>
                )}
              </NavLink>
            ))}
          </Fragment>
        ))}
      </nav>

      <div className="xl-side__foot" style={{ padding: 8 }}>
        <label
          htmlFor={SIDEBAR_COLLAPSE_ID}
          className="xl-nav__item xl-collapse-btn"
          title="Collapse sidebar"
          style={{ width: "100%" }}
        >
          <Icon name="chevron" className="xl-collapse-ico" />
          <span className="xl-collapse-lbl">Collapse</span>
        </label>
      </div>
    </aside>
  );
}
