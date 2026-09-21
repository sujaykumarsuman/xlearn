// Shared loading / error / empty states (S12 hardening). Every screen composes these
// so the three data states look and behave identically and stay on the design system —
// no ad-hoc per-screen markup. They reuse theme.css/app.css classes verbatim.
import type { ReactNode } from "react";
import { Icon, type IconName } from "./Icon";

/**
 * Spinner is a VISIBLE loading indicator: the DS rotation (`.ds-spin`) applied to the
 * refresh glyph, with its label announced to assistive tech via role="status". (The
 * bare `.ds-spin` on an empty element renders nothing — it needs a glyph to spin.)
 */
export function Spinner({ label = "Loading…", size }: { label?: string; size?: "sm" | "lg" }) {
  const sizeCls = size === "sm" ? " xl-ico--sm" : size === "lg" ? " xl-ico--lg" : "";
  return (
    <span className="xl-spinner" role="status">
      <Icon name="refresh" className={`ds-spin${sizeCls}`} />
      <span>{label}</span>
    </span>
  );
}

/** LoadingState is a Spinner given its own padded row (the common list-loading slot). */
export function LoadingState({ label }: { label?: string }) {
  return (
    <div style={{ padding: "24px 0" }}>
      <Spinner label={label} />
    </div>
  );
}

/**
 * ErrorState is the DS error surface: an alert panel with an optional Retry. role="alert"
 * so screen readers announce it when it appears.
 */
export function ErrorState({
  message = "Something went wrong.",
  onRetry,
}: {
  message?: string;
  onRetry?: () => void;
}) {
  return (
    <div
      className="xl-panel"
      role="alert"
      style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}
    >
      <Icon name="alert" />
      <span className="xl-mut" style={{ flex: 1 }}>
        {message}
      </span>
      {onRetry && (
        <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  );
}

/**
 * EmptyState is the calm "nothing here yet" block: a muted glyph, a title, and optional
 * supporting copy. role="status" so it is announced when a list resolves to empty.
 */
export function EmptyState({
  icon = "check",
  title,
  children,
}: {
  icon?: IconName;
  title: string;
  children?: ReactNode;
}) {
  return (
    <div className="xl-state" role="status">
      <Icon name={icon} className="xl-ico--lg" />
      <div className="xl-state__title">{title}</div>
      {children && <p style={{ margin: 0, maxWidth: 440 }}>{children}</p>}
    </div>
  );
}
