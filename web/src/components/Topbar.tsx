import { Fragment, useEffect, useRef, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useLogout, useMe } from "../lib/auth";
import { buildCrumbs } from "../nav";
import { CoachModelSwitcher } from "./CoachModelSwitcher";
import { Icon } from "./Icon";
import { PathSwitcher } from "./PathSwitcher";

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

/** The brand lockup, shown at the top-left on shells without a sidebar (home /
 *  Settings) so xLearn is still identified there. Links back to the Catalog. */
function BrandInline() {
  return (
    <Link to="/" className="xl-topbar__brand" aria-label="xLearn — all paths">
      <span className="xl-brand__mark" style={{ width: 26, height: 26, fontSize: 14 }}>
        x
      </span>
      <span className="xl-brand__name">
        x<b>Learn</b>
      </span>
    </Link>
  );
}

/**
 * The top bar (F001): breadcrumb + curriculum selector inside a curriculum
 * (`variant:"curriculum"`), or the brand lockup on the sidebar-less home / Settings
 * (`variant:"plain"`). The ⌘K search is gone — the selector took its place. Both
 * variants carry the notifications bell + account menu.
 */
export function Topbar({ variant, slug }: { variant: "plain" | "curriculum"; slug?: string }) {
  const me = useMe();
  const navigate = useNavigate();
  const logout = useLogout();

  const name = me.data?.account.display_name ?? "Account";
  const email = me.data?.account.email ?? "";
  const initial = name.trim().charAt(0).toUpperCase() || "?";

  // Account menu as an accessible button-disclosure (replaces the pure-CSS checkbox,
  // which announced as "checkbox" and had no visible keyboard focus). Escape and an
  // outside click close it.
  const [acctOpen, setAcctOpen] = useState(false);
  const acctRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!acctOpen) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setAcctOpen(false);
    };
    const onPointer = (e: PointerEvent) => {
      if (acctRef.current && !acctRef.current.contains(e.target as Node)) setAcctOpen(false);
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [acctOpen]);
  const closeAcct = () => setAcctOpen(false);

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
      {variant === "curriculum" ? <Crumb /> : <BrandInline />}

      <div className="xl-topbar__spacer" />

      {variant === "curriculum" && slug && <PathSwitcher slug={slug} />}

      <CoachModelSwitcher />

      <button className="ds-iconbtn" aria-label="Notifications">
        <Icon name="bell" />
      </button>

      <div className="xl-acct" ref={acctRef}>
        <button
          type="button"
          className="ds-avatar xl-topbar__avatar xl-acct-btn"
          aria-haspopup="true"
          aria-expanded={acctOpen}
          aria-label="Account menu"
          style={{ border: "none", padding: 0, cursor: "pointer" }}
          onClick={() => setAcctOpen((v) => !v)}
        >
          {initial}
        </button>
        {acctOpen && (
          <>
            <button
              type="button"
              className="xl-acct-backdrop"
              aria-label="Close account menu"
              tabIndex={-1}
              style={{ display: "block", background: "none", border: "none", padding: 0 }}
              onClick={closeAcct}
            />
            <div className="xl-acct-menu" style={{ display: "block" }}>
              <div className="xl-acct-head">
                <span className="ds-avatar" style={{ width: 36, height: 36, fontSize: 14 }}>
                  {initial}
                </span>
                <div>
                  <b>{name}</b>
                  {email && <span>{email}</span>}
                </div>
              </div>
              <Link to="/settings" className="xl-acct-item" onClick={closeAcct}>
                <Icon name="settings" className="xl-ico--sm" /> Settings
              </Link>
              <Link to="/dsa/progress" className="xl-acct-item" onClick={closeAcct}>
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
          </>
        )}
      </div>
    </div>
  );
}
