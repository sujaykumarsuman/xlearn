import type { ReactNode } from "react";
import type { IconName } from "./Icon";
import { Icon } from "./Icon";

interface PlaceholderProps {
  /** Small mono eyebrow above the title. */
  eyebrow: string;
  /** Screen title (H1). */
  title: string;
  /** Icon shown in the stub panel. */
  icon: IconName;
  /** Sprint tag, e.g. "S09". */
  sprint: string;
  /** One-line summary of what this screen will do when built. */
  summary: string;
  /** Optional extra body (notes/lists). */
  children?: ReactNode;
}

/**
 * Placeholder renders the real page chrome (eyebrow + title) plus an on-brand
 * "coming in Sprint NN" stub body. Every route uses it this sprint until its
 * feature sprint fills the body.
 */
export function Placeholder({ eyebrow, title, icon, sprint, summary, children }: PlaceholderProps) {
  return (
    <>
      <div className="xl-page-h">
        <div>
          <div className="xl-eyebrow">{eyebrow}</div>
          <h1 style={{ marginTop: 6 }}>{title}</h1>
          <p>{summary}</p>
        </div>
        <span className="xl-lock">
          <Icon name="lock" className="xl-ico--sm" />
          Coming in Sprint {sprint.replace(/^S/, "")}
        </span>
      </div>

      <div className="xl-panel">
        <div className="xl-panel__h">
          <Icon name={icon} className="" />
          <h3 style={{ flex: 1 }}>{title}</h3>
          <span className="ds-badge ds-badge--violet ds-mono">{sprint}</span>
        </div>
        <div className="xl-panel__b">
          <p className="xl-dim" style={{ fontSize: 13.5, lineHeight: 1.6 }}>{summary}</p>
          {children}
          <div
            style={{
              marginTop: 16,
              display: "flex",
              alignItems: "center",
              gap: 10,
              padding: "12px 14px",
              borderRadius: 9,
              background: "rgba(155,140,240,.06)",
              border: "1px dashed var(--ds-line-2)",
            }}
          >
            <Icon name="layers" className="xl-ico--sm" />
            <span className="xl-mut" style={{ fontSize: 12.5 }}>
              This screen is a scaffold. Its interactive body ships in Sprint {sprint.replace(/^S/, "")}.
            </span>
          </div>
        </div>
      </div>
    </>
  );
}
