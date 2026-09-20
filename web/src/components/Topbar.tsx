import { Fragment } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useLogout, useMe } from "../lib/auth";
import { buildCrumbs } from "../nav";
import { Icon } from "./Icon";

// A single top-bar instance mounts at a time, so a fixed id is safe for the
// pure-CSS account-menu toggle.
const ACCT_ID = "xl-acct";

/** Breadcrumb derived from the app-relative path + query (basename already
 *  stripped). The trail — and the Week/Concept special cases — live in buildCrumbs. */
function Crumb() {
  const { pathname, search } = useLocation();
  const crumbs = buildCrumbs(pathname, search);

  return (
    <div className="xl-crumb">
      {crumbs.map((c, i) => {
        const last = i === crumbs.length - 1;
        return (
          <Fragment key={`${c.label}-${i}`}>
            {i > 0 && <span className="xl-crumb__sep">/</span>}
            {last || !c.to ? <b>{c.label}</b> : <Link to={c.to}>{c.label}</Link>}
          </Fragment>
        );
      })}
    </div>
  );
}

/** The top bar: breadcrumb, cmd-K search pill, notifications, account menu. */
export function Topbar() {
  const me = useMe();
  const navigate = useNavigate();
  const logout = useLogout();

  const name = me.data?.account.display_name ?? "Account";
  const email = me.data?.account.email ?? "";
  const initial = name.trim().charAt(0).toUpperCase() || "?";

  // Navigate to /auth only on a successful revocation. A failed logout must NOT
  // bounce the user to /auth (the still-cached, still-valid session would just send
  // them back into the app) — keep them here and show the error instead.
  const signOut = () => {
    logout.mutate(undefined, {
      onSuccess: () => navigate("/auth", { replace: true }),
    });
  };

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
          {initial}
        </label>
        <label htmlFor={ACCT_ID} className="xl-acct-backdrop" aria-hidden="true" />
        <div className="xl-acct-menu">
          <div className="xl-acct-head">
            <span className="ds-avatar" style={{ width: 36, height: 36, fontSize: 14 }}>
              {initial}
            </span>
            <div>
              <b>{name}</b>
              {email && <span>{email}</span>}
            </div>
          </div>
          <Link to="/settings" className="xl-acct-item">
            <Icon name="settings" className="xl-ico--sm" /> Settings
          </Link>
          <Link to="/dsa/progress" className="xl-acct-item">
            <Icon name="chart" className="xl-ico--sm" /> Progress &amp; stats
          </Link>
          <div className="xl-acct-sep" />
          <button
            type="button"
            className="xl-acct-item xl-acct-item--danger"
            onClick={signOut}
            disabled={logout.isPending}
          >
            <Icon name="signout" className="xl-ico--sm" /> {logout.isPending ? "Signing out…" : "Sign out"}
          </button>
          {logout.isError && (
            <div role="alert" className="xl-mut" style={{ padding: "6px 12px", fontSize: 11.5, color: "var(--ds-err)" }}>
              Couldn’t sign out — try again.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
