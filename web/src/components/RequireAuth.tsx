import { Navigate, Outlet, useLocation } from "react-router-dom";
import { useMe } from "../lib/auth";
import { firstSegment, isCourseCandidate } from "../lib/course";
import { usePaths } from "../lib/curriculum";
import { FullScreen, Spinner } from "./States";

/**
 * AuthedShell gates the app: it reads GET /me and renders the nested routes (each of
 * which picks its own layout — PlainShell or CurriculumShell) only when the session is
 * valid AND onboarding is complete (ADR-0006; S10 route gating). An unauthenticated
 * response (401) or an account with onboarding still incomplete (completed_at IS NULL)
 * redirects to the standalone /auth screen, which begins OAuth or resumes onboarding at
 * the first unfinished step. /auth lives outside this gate, so there is no redirect loop.
 */
export default function AuthedShell() {
  const me = useMe();
  const { pathname } = useLocation();
  // A course route resolves its course against the catalog (CurriculumShell, useCourse):
  // start that fetch alongside GET /me rather than after it, so a course page doesn't pay
  // a second round trip before its frame renders. Same query, so it is fetched once.
  usePaths(isCourseCandidate(firstSegment(pathname)));

  if (me.isLoading) {
    return (
      <FullScreen>
        <Spinner label="Loading…" />
      </FullScreen>
    );
  }
  if (me.isError) {
    if (me.error.isUnauthenticated) {
      return <Navigate to="/auth" replace />;
    }
    return (
      <FullScreen>
        <span className="xl-mut">Couldn’t reach xLearn.</span>
        <button type="button" className="ds-btn ds-btn--secondary" onClick={() => me.refetch()}>
          Retry
        </button>
      </FullScreen>
    );
  }
  if (!me.data) {
    return <Navigate to="/auth" replace />;
  }
  // First-run gate: an authenticated account whose onboarding is not complete belongs
  // in the flow (path → budget → key), not the app. /auth resumes at the first
  // unfinished step and routes away once complete.
  if (!me.data.onboarding.completed) {
    return <Navigate to="/auth" replace />;
  }
  return <Outlet />;
}
