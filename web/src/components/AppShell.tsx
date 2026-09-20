import { Outlet } from "react-router-dom";
import { SIDEBAR_COLLAPSE_ID } from "../nav";
import { Coach } from "./Coach";
import { IconSprite } from "./Icon";
import { Sidebar } from "./Sidebar";
import { Topbar } from "./Topbar";

/**
 * AppShell is the persistent xLearn frame (ADR-0008): sidebar + top bar + coach
 * around the routed screen. The collapse checkbox is the first child of
 * `.xl-app` so the pure-CSS `.xl-collapse-cb:checked ~ .xl-side` rule in
 * theme.css can collapse the sidebar to an icon rail.
 */
export default function AppShell() {
  return (
    <div className="xl-app">
      <input
        type="checkbox"
        id={SIDEBAR_COLLAPSE_ID}
        className="xl-collapse-cb"
        aria-label="Collapse sidebar"
      />
      <IconSprite />
      <Sidebar />
      <div className="xl-main">
        <Topbar />
        <main className="xl-content xl-scroll">
          <Outlet />
        </main>
      </div>
      <Coach />
    </div>
  );
}
