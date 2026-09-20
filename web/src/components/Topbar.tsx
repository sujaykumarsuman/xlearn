import { Fragment } from "react";
import { Link, useLocation } from "react-router-dom";
import { Icon } from "./Icon";

// A single top-bar instance mounts at a time, so a fixed id is safe for the
// pure-CSS account-menu toggle.
const ACCT_ID = "xl-acct";

/** Breadcrumb derived from the app-relative path (basename already stripped). */
function Crumb() {
  const { pathname } = useLocation();
  const segs = pathname.replace(/\/+$/, "").split("/").filter(Boolean);

  return (
    <div className="xl-crumb">
      {segs.length === 0 ? <b>xlearn</b> : <Link to="/">xlearn</Link>}
      {segs.map((seg, i) => {
        const to = "/" + segs.slice(0, i + 1).join("/");
        const last = i === segs.length - 1;
        return (
          <Fragment key={to}>
            <span className="xl-crumb__sep">/</span>
            {last ? <b>{seg}</b> : <Link to={to}>{seg}</Link>}
          </Fragment>
        );
      })}
    </div>
  );
}

/** The top bar: breadcrumb, cmd-K search pill, notifications, account menu. */
export function Topbar() {
  return (
    <div className="xl-topbar">
      <Crumb />

      <button className="xl-cmdk" disabled title="Command palette — coming soon">
        <Icon name="search" className="xl-ico--sm" />
        <span style={{ flex: 1, textAlign: "left" }}>Search or jump to…</span>
        <span className="xl-kbd">⌘K</span>
      </button>

      <button className="ds-iconbtn" aria-label="Notifications">
        <Icon name="bell" />
      </button>

      <div className="xl-acct">
        <input type="checkbox" id={ACCT_ID} className="xl-acct-cb" aria-label="Account menu" />
        <label htmlFor={ACCT_ID} className="ds-avatar xl-topbar__avatar xl-acct-btn" title="Account">
          S
        </label>
        <label htmlFor={ACCT_ID} className="xl-acct-backdrop" aria-hidden="true" />
        <div className="xl-acct-menu">
          <div className="xl-acct-head">
            <span className="ds-avatar" style={{ width: 36, height: 36, fontSize: 14 }}>
              S
            </span>
            <div>
              <b>Sujay Kumar</b>
              <span>sujaykumar.dev@gmail.com</span>
            </div>
          </div>
          <Link to="/settings" className="xl-acct-item">
            <Icon name="settings" className="xl-ico--sm" /> Settings
          </Link>
          <Link to="/dsa/progress" className="xl-acct-item">
            <Icon name="chart" className="xl-ico--sm" /> Progress &amp; stats
          </Link>
          <div className="xl-acct-sep" />
          <Link to="/auth" className="xl-acct-item xl-acct-item--danger">
            <Icon name="signout" className="xl-ico--sm" /> Sign out
          </Link>
        </div>
      </div>
    </div>
  );
}
