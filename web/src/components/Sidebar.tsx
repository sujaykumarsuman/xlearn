import { Fragment, useState } from "react";
import { Link, NavLink } from "react-router-dom";
import { NAV, SIDEBAR_COLLAPSE_ID } from "../nav";
import { Icon } from "./Icon";

/** The collapsible left sidebar: brand, path switcher, nav, and the pure-CSS
 *  collapse toggle (label bound to the checkbox rendered by AppShell). */
export function Sidebar() {
  const [pathOpen, setPathOpen] = useState(false);

  return (
    <aside className="xl-side">
      <div className="xl-brand">
        <div className="xl-brand__mark">x</div>
        <div className="xl-brand__name">
          x<b>Learn</b>
        </div>
      </div>

      <div className="xl-pathsw">
        <button
          className="xl-pathsw__btn"
          onClick={() => setPathOpen((o) => !o)}
          aria-expanded={pathOpen}
          aria-haspopup="true"
        >
          <span className="xl-pathsw__ic">DSA</span>
          <span style={{ flex: 1 }}>
            <span className="xl-pathsw__t">DSA Interview Mastery</span>
            <span className="xl-pathsw__s">16 weeks · 151 problems</span>
          </span>
          <Icon name="chevdown" className="xl-ico--sm" />
        </button>
      </div>
      {pathOpen && (
        <div className="xl-pathsw__menu" role="menu">
          <div className="xl-pathsw__opt xl-pathsw__opt--active">
            <span className="ds-dot ds-dot--ok" /> DSA Interview Mastery
          </div>
          <div className="xl-pathsw__opt xl-pathsw__opt--locked">
            <Icon name="lock" className="xl-ico--sm" /> System Design · <i>Coming soon</i>
          </div>
          <div className="xl-pathsw__opt xl-pathsw__opt--locked">
            <Icon name="lock" className="xl-ico--sm" /> Go Concurrency · <i>Coming soon</i>
          </div>
          <Link
            className="xl-pathsw__opt"
            to="/"
            style={{ color: "var(--ds-dim)" }}
            onClick={() => setPathOpen(false)}
          >
            <Icon name="grid" className="xl-ico--sm" /> Browse all paths
          </Link>
        </div>
      )}

      <nav className="xl-nav">
        {NAV.map((section, i) => (
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
