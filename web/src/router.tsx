import { createBrowserRouter, type RouteObject } from "react-router-dom";
import AppShell from "./components/AppShell";
import Auth from "./screens/Auth";
import Catalog from "./screens/Catalog";
import Concept from "./screens/Concept";
import Dashboard from "./screens/Dashboard";
import Mistakes from "./screens/Mistakes";
import Mock from "./screens/Mock";
import NotFound from "./screens/NotFound";
import Problem from "./screens/Problem";
import Progress from "./screens/Progress";
import Revision from "./screens/Revision";
import Roadmap from "./screens/Roadmap";
import Settings from "./screens/Settings";
import Week from "./screens/Week";

// All routes are app-relative; the router prepends the /xlearn basename. The
// 12 screens from the design system, plus a catch-all 404, render inside the
// persistent AppShell. Route → screen → fills-in sprint:
//   /                    Catalog     (S03)
//   /dsa                 Roadmap     (S03)
//   /dsa/dashboard       Dashboard   (S09)
//   /dsa/week/:n         Week        (S04)
//   /dsa/concept/:slug   Concept     (S04)
//   /dsa/problem/:id     Problem     (S05)
//   /dsa/revision        Revision    (S06)
//   /dsa/mistakes        Mistakes    (S07)
//   /dsa/mock            Mock        (S08)
//   /dsa/progress        Progress    (S09)
//   /settings            Settings    (S10)
//   /auth                Auth        (S02)
export const routes: RouteObject[] = [
  {
    path: "/",
    element: <AppShell />,
    children: [
      { index: true, element: <Catalog /> },
      { path: "dsa", element: <Roadmap /> },
      { path: "dsa/dashboard", element: <Dashboard /> },
      { path: "dsa/week/:n", element: <Week /> },
      { path: "dsa/concept/:slug", element: <Concept /> },
      { path: "dsa/problem/:id", element: <Problem /> },
      { path: "dsa/revision", element: <Revision /> },
      { path: "dsa/mistakes", element: <Mistakes /> },
      { path: "dsa/mock", element: <Mock /> },
      { path: "dsa/progress", element: <Progress /> },
      { path: "settings", element: <Settings /> },
      { path: "auth", element: <Auth /> },
      { path: "*", element: <NotFound /> },
    ],
  },
];

export const router = createBrowserRouter(routes, { basename: "/xlearn" });
