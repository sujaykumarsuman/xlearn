import { Navigate } from "react-router-dom";
import { useMe } from "../lib/auth";
import AppShell from "./AppShell";

/**
 * AuthedShell gates the app: it reads GET /me and renders the AppShell only when
 * the session is valid AND onboarding step 1 is done (ADR-0006; prompt-s02 route
 * gating). An unauthenticated response (401) or an account that hasn't picked a
 * path yet redirects to the standalone /auth screen, which begins OAuth or resumes
 * onboarding. /auth lives outside this gate, so there is no redirect loop.
 */
export default function AuthedShell() {
  const me = useMe();

  if (me.isLoading) {
    return (
      <FullScreen>
        <span className="ds-spin" aria-hidden="true" />
        <span>Loading…</span>
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
  // First-run gate: an authenticated account that hasn't chosen a path yet belongs
  // in onboarding, not the app (its screens assume a chosen path).
  if (!me.data.onboarding.path_chosen) {
    return <Navigate to="/auth" replace />;
  }
  return <AppShell />;
}

/** FullScreen centers a small status message on the app background. */
function FullScreen({ children }: { children: React.ReactNode }) {
  return (
    <div
      style={{
        minHeight: "100vh",
        display: "grid",
        placeItems: "center",
        background: "var(--ds-bg)",
        color: "var(--ds-dim)",
        gap: 12,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 10 }}>{children}</div>
    </div>
  );
}
