import type { ReactNode } from "react";
import { Outlet, useMatches } from "react-router-dom";
import { isCourseNotFoundHandle, useCourse } from "../lib/course";
import { SIDEBAR_COLLAPSE_ID } from "../nav";
import ComingSoon from "../screens/ComingSoon";
import NotFound from "../screens/NotFound";
import { Coach } from "./Coach";
import { IconSprite } from "./Icon";
import { Sidebar } from "./Sidebar";
import { FullScreen, Spinner } from "./States";
import { Topbar } from "./Topbar";

/**
 * The xLearn app frame comes in two variants that share the top bar + coach (F001):
 *
 *  - CurriculumShell — the frame for the one `/:course/*` subtree (ADR-0026 §5). It
 *    resolves the URL's course against the session-gated catalog (useCourse) and renders:
 *      · active      → the full frame WITH the sidebar (the manifest's nav) and the
 *                      course switcher, and the routed screen;
 *      · coming_soon → the coming-soon teaser in the plain frame, for every sub-route
 *                      (AB02-F4; no sidebar, no switcher, no data calls);
 *      · unknown     → v1's NotFound in the plain frame (AB02-F6), as does an unknown
 *                      sub-route of an active course;
 *      · loading / error → the app-level full-screen state, so neither frame flashes
 *                      before the course is known and no course data call fires early.
 *  - PlainShell — sidebar-less, for the hub-level pages: the Catalog home (a bare list
 *    of paths to pick) and account-level Settings.
 *
 * Both frames keep .xl-app position:relative + overflow:hidden (theme.css), so the coach
 * FAB anchors correctly and the routed screen scrolls inside .xl-content.
 */
export function CurriculumShell() {
  const course = useCourse();
  // An unknown sub-route of the course (/<course>/bogus) matched the subtree's catch-all.
  const subRouteUnknown = useMatches().some((m) => isCourseNotFoundHandle(m.handle));

  switch (course.state) {
    case "loading":
      return (
        <FullScreen>
          <Spinner label="Loading…" />
        </FullScreen>
      );
    case "error":
      return (
        <FullScreen>
          <span className="xl-mut">Couldn’t reach xLearn.</span>
          <button type="button" className="ds-btn ds-btn--secondary" onClick={course.retry}>
            Retry
          </button>
        </FullScreen>
      );
    case "unknown":
      return (
        <PlainFrame>
          <NotFound />
        </PlainFrame>
      );
    case "coming_soon":
      return <PlainFrame>{course.path && <ComingSoon path={course.path} />}</PlainFrame>;
  }

  if (subRouteUnknown) {
    return (
      <PlainFrame>
        <NotFound />
      </PlainFrame>
    );
  }

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
      <Sidebar slug={course.slug} nav={course.view?.nav} />
      <div className="xl-main">
        <Topbar variant="curriculum" slug={course.slug} />
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
    <PlainFrame>
      <Outlet />
    </PlainFrame>
  );
}

/** PlainFrame is the sidebar-less frame: brand lockup top bar, the content, the coach. */
function PlainFrame({ children }: { children: ReactNode }) {
  return (
    <div className="xl-app">
      <IconSprite />
      <div className="xl-main">
        <Topbar variant="plain" />
        <main className="xl-content xl-scroll">{children}</main>
      </div>
      <Coach />
    </div>
  );
}
