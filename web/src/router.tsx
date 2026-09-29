import { createBrowserRouter, type RouteObject } from "react-router-dom";
import { CurriculumShell, PlainShell } from "./components/AppShell";
import AuthedShell from "./components/RequireAuth";
import { COURSE_NOT_FOUND_HANDLE } from "./lib/course";
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
//   CurriculumShell (+ sidebar) — inside a course: /:course/*.
//
// Route → screen → fills-in sprint:
//   /                        Catalog     (S03; F002 enrollment-aware)
//   /:course                 Roadmap     (S03)
//   /:course/dashboard       Dashboard   (S07 weak-area + reviews-due cards; full plan/stats S09)
//   /:course/problems        Problems    (the practice arena)
//   /:course/week/:n         Week        (S04)
//   /:course/concept/:slug   Concept     (S04)
//   /:course/problem/:id     Problem     (S05)
//   /:course/revision        Revision    (S06)
//   /:course/mistakes        Mistakes    (S07)
//   /:course/mock            Mock        (S08)
//   /:course/progress        Progress    (S09)
//   /settings                Settings    (S10)
//   /auth                    Auth        (S02, standalone; username is claimed here in onboarding)
//   /u/:username             UserDashboard (F009; PUBLIC — outside AuthedShell, like /auth)
//
// Courses (m1-03, ADR-0026 §5): ONE dynamic `:course` subtree, the same segments for
// every course. CurriculumShell resolves the slug against the session-gated catalog: an
// active course renders its screens, a coming_soon course the teaser, anything else
// NotFound. Static segments outrank the dynamic one (/settings, /auth, /u/…, tested),
// and the course-slug guard keeps every static first segment out of the slug space.
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
          {
            path: ":course",
            children: [
              { index: true, element: <Roadmap /> },
              { path: "problems", element: <Problems /> },
              { path: "dashboard", element: <Dashboard /> },
              { path: "week/:n", element: <Week /> },
              { path: "concept/:slug", element: <Concept /> },
              { path: "problem/:id", element: <Problem /> },
              { path: "revision", element: <Revision /> },
              { path: "mistakes", element: <Mistakes /> },
              { path: "mock", element: <Mock /> },
              { path: "progress", element: <Progress /> },
              // An unknown sub-route of a course: NotFound in the plain frame, as in v1
              // (CurriculumShell reads the handle).
              { path: "*", element: <NotFound />, handle: COURSE_NOT_FOUND_HANDLE },
            ],
          },
        ],
      },
    ],
  },
];

export const router = createBrowserRouter(routes, { basename: "/xlearn" });
