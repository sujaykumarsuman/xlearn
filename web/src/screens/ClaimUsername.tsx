import { useEffect, useState } from "react";
import { Navigate, useNavigate } from "react-router-dom";
import { Icon } from "../components/Icon";
import { useMe, useSetUsername, useUsernameAvailability } from "../lib/auth";

/**
 * ClaimUsername (`/xlearn/claim-username`, F009): the claim gate the avatar → Dashboard link
 * routes to when the signed-in account has no username yet. It's a focused first-time claim
 * (the canonical field also lives in Settings). On success it navigates to the freshly public
 * dashboard at /xlearn/<username>. Authed route (inside AuthedShell).
 */
export default function ClaimUsername() {
  const me = useMe();
  const navigate = useNavigate();
  const setU = useSetUsername();
  const [value, setValue] = useState("");
  const [debounced, setDebounced] = useState("");

  // Debounce the availability probe so it doesn't fire on every keystroke.
  useEffect(() => {
    const t = setTimeout(() => setDebounced(value.trim().toLowerCase()), 350);
    return () => clearTimeout(t);
  }, [value]);
  const avail = useUsernameAvailability(debounced);

  // Already claimed → straight to the public dashboard (no re-claiming here).
  const existing = me.data?.account.username;
  if (existing) return <Navigate to={`/${existing}`} replace />;

  const normalized = value.trim().toLowerCase();
  const canSubmit = normalized.length >= 3 && avail.data?.available === true && !setU.isPending;

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canSubmit) return;
    setU.mutate(normalized, {
      onSuccess: (r) => navigate(`/${r.username}`, { replace: true }),
    });
  };

  return (
    <div style={{ maxWidth: 520, margin: "40px auto 0" }}>
      <div className="xl-page-h" style={{ marginBottom: 20 }}>
        <div className="xl-eyebrow">Public profile</div>
        <h1 style={{ marginTop: 6, fontSize: 24, fontWeight: 700, letterSpacing: "-.3px" }}>Claim your username</h1>
        <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)" }}>
          Your username is the address of your public dashboard —{" "}
          <span className="ds-mono" style={{ color: "var(--ds-dim)" }}>projects.sujaykumar.dev/xlearn/&lt;you&gt;</span>. You can
          also sign in with it.
        </p>
      </div>

      <form className="xl-panel" onSubmit={submit}>
        <div className="xl-panel__b" style={{ display: "flex", flexDirection: "column", gap: 6 }}>
          <div className="ds-field">
            <label className="ds-field__label" htmlFor="username">Username</label>
            <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <span className="ds-mono" style={{ color: "var(--ds-muted)" }}>/xlearn/</span>
              <input
                id="username"
                className="ds-input"
                style={{ flex: 1 }}
                autoFocus
                autoComplete="off"
                autoCapitalize="none"
                spellCheck={false}
                placeholder="your-handle"
                value={value}
                onChange={(e) => setValue(e.target.value)}
              />
            </div>
          </div>
          <UsernameHint value={normalized} avail={avail} />
          {setU.isError && (
            <div role="alert" style={{ fontSize: 12.5, color: "var(--ds-err)" }}>
              {setU.error.message}
            </div>
          )}
          <div style={{ marginTop: 8 }}>
            <button type="submit" className="ds-btn ds-btn--primary" disabled={!canSubmit}>
              {setU.isPending ? "Claiming…" : "Claim username"}
            </button>
          </div>
        </div>
      </form>
    </div>
  );
}

/** The live validity/availability line beneath the field. */
function UsernameHint({
  value,
  avail,
}: {
  value: string;
  avail: ReturnType<typeof useUsernameAvailability>;
}) {
  if (value.length === 0) {
    return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>3–30 chars · lowercase letters, numbers, hyphens.</span>;
  }
  if (value.length < 3) {
    return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>Keep going — at least 3 characters.</span>;
  }
  if (avail.isLoading) {
    return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>Checking availability…</span>;
  }
  if (avail.data?.available) {
    return (
      <span style={{ fontSize: 12, color: "var(--ds-ok)" }}>
        <Icon name="check" className="xl-ico--sm" /> @{value} is available
      </span>
    );
  }
  if (avail.data && !avail.data.available) {
    return <span style={{ fontSize: 12, color: "var(--ds-err)" }}>{avail.data.reason ?? "That username isn’t available."}</span>;
  }
  return null;
}
