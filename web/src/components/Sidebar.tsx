import { Fragment } from "react";
import { NavLink } from "react-router-dom";
import { navForPath, SIDEBAR_COLLAPSE_ID } from "../nav";
import { Icon } from "./Icon";

/**
 * The collapsible left sidebar (F001): brand, the curriculum-scoped nav, and the
 * pure-CSS collapse toggle. It renders only inside a curriculum; the path switcher
 * moved to the top bar, so the sidebar is now purely this path's navigation.
 */
export function Sidebar({ slug }: { slug: string }) {
  const nav = navForPath(slug);

  return (
    <aside className="xl-side">
      <div className="xl-brand">
        <div className="xl-brand__mark">x</div>
        <div className="xl-brand__name">
          x<b>Learn</b>
        </div>
      </div>

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
