import { createBrowserRouter, type RouteObject } from "react-router-dom";
import { CurriculumShell, PlainShell } from "./components/AppShell";
import AuthedShell from "./components/RequireAuth";
import Auth from "./screens/Auth";
import Catalog from "./screens/Catalog";
import Concept from "./screens/Concept";
import Dashboard from "./screens/Dashboard";
import Mistakes from "./screens/Mistakes";
import Mock from "./screens/Mock";
import NotFound from "./screens/NotFound";
import Problem from "./screens/Problem";
import Problems from "./screens/Problems";
import Progress from "./screens/Progress";
import Revision from "./screens/Revision";
import Roadmap from "./screens/Roadmap";
import Settings from "./screens/Settings";
import UserDashboard from "./screens/UserDashboard";
import Week from "./screens/Week";

// All routes are app-relative; the router prepends the /xlearn basename.
//
// /auth is a STANDALONE pre-auth screen (no app shell): OAuth sign-in + the 3-step
// onboarding (ADR-0006). Every other route is gated by AuthedShell (an unauthenticated
// GET /me redirects to /auth, S02), then picks one of two layouts (F001):
//   PlainShell (no sidebar)     — hub level: Catalog home + Settings.
//   CurriculumShell (+ sidebar) — inside a curriculum: /dsa/*.
//
// Route → screen → fills-in sprint:
//   /                    Catalog     (S03; F002 enrollment-aware)
//   /dsa                 Roadmap     (S03)
//   /dsa/dashboard       Dashboard   (S07 weak-area + reviews-due cards; full plan/stats S09)
//   /dsa/week/:n         Week        (S04)
//   /dsa/concept/:slug   Concept     (S04)
//   /dsa/problem/:id     Problem     (S05)
//   /dsa/revision        Revision    (S06)
//   /dsa/mistakes        Mistakes    (S07)
//   /dsa/mock            Mock        (S08)
//   /dsa/progress        Progress    (S09)
//   /settings            Settings    (S10)
//   /auth                Auth        (S02, standalone; username is claimed here in onboarding)
//   /u/:username         UserDashboard (F009; PUBLIC — outside AuthedShell, like /auth)
//
// Routing note (ADR-0025): the public profile lives under its own /u/ prefix, OUTSIDE
// AuthedShell, so usernames never share a namespace with app routes — a new course or
// top-level route can't shadow a profile, and an unknown /xlearn/<x> is the in-shell
// NotFound. (F009 first shipped bare /xlearn/<username>; that route was dropped.)
export const routes: RouteObject[] = [
  { path: "/auth", element: <Auth /> },
  { path: "/u/:username", element: <UserDashboard /> },
  {
    path: "/",
    element: <AuthedShell />,
    children: [
      {
        element: <PlainShell />,
        children: [
          { index: true, element: <Catalog /> },
          { path: "settings", element: <Settings /> },
          { path: "*", element: <NotFound /> },
        ],
      },
      {
        element: <CurriculumShell />,
        children: [
          { path: "dsa", element: <Roadmap /> },
          { path: "dsa/problems", element: <Problems /> },
          { path: "dsa/dashboard", element: <Dashboard /> },
          { path: "dsa/week/:n", element: <Week /> },
          { path: "dsa/concept/:slug", element: <Concept /> },
          { path: "dsa/problem/:id", element: <Problem /> },
          { path: "dsa/revision", element: <Revision /> },
          { path: "dsa/mistakes", element: <Mistakes /> },
          { path: "dsa/mock", element: <Mock /> },
          { path: "dsa/progress", element: <Progress /> },
        ],
      },
    ],
  },
];

export const router = createBrowserRouter(routes, { basename: "/xlearn" });
