import { Outlet, useLocation } from "react-router-dom";
import { SIDEBAR_COLLAPSE_ID } from "../nav";
import { Coach } from "./Coach";
import { IconSprite } from "./Icon";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

/**
 * The xLearn app frame comes in two variants that share the top bar + coach (F001):
 *
 *  - CurriculumShell — the full frame WITH the left sidebar, for routes inside a
 *    curriculum (/dsa/*). The nav is curriculum-scoped, so the sidebar belongs here,
 *    not on the hub-level pages.
 *  - PlainShell — sidebar-less, for the hub-level pages: the Catalog home (a bare list
 *    of paths to pick) and account-level Settings.
 *
 * Both keep .xl-app position:relative + overflow:hidden (theme.css), so the coach FAB
 * anchors correctly and the routed screen scrolls inside .xl-content.
 */
export function CurriculumShell() {
  const { pathname } = useLocation();
  // Every curriculum route is /{slug}/... — the first segment is the active path.
  const slug = pathname.split("/").filter(Boolean)[0] ?? "dsa";
  return (
    <div className="xl-app">
      {/* First child of .xl-app so the pure-CSS `.xl-collapse-cb:checked ~ .xl-side`
          rule in theme.css can collapse the sidebar to an icon rail. */}
      <input
        type="checkbox"
        id={SIDEBAR_COLLAPSE_ID}
        className="xl-collapse-cb"
        aria-label="Collapse sidebar"
      />
      <IconSprite />
      <Sidebar slug={slug} />
      <div className="xl-main">
        <Topbar variant="curriculum" slug={slug} />
        <main className="xl-content xl-scroll">
          <Outlet />
        </main>
      </div>
      <Coach />
    </div>
  );
}

export function PlainShell() {
  return (
    <div className="xl-app">
      <IconSprite />
      <div className="xl-main">
        <Topbar variant="plain" />
        <main className="xl-content xl-scroll">
          <Outlet />
        </main>
      </div>
      <Coach />
    </div>
  );
}
