import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { courseIcon } from "../lib/courseIcons";
import type { Path } from "../lib/curriculum";

const DIM_TEXT = { fontSize: 13.5, color: "var(--ds-dim)", lineHeight: 1.6 } as const;

/**
 * ComingSoon is the teaser for a `coming_soon` course (AB02-F4): every
 * /xlearn/<course>/* sub-route renders it in the plain frame (no sidebar, no course
 * switcher, no data calls — the course's data routes 404 until it opens). Title, summary,
 * totals and icon come from the catalog entry (v1's ComingSoonCard content), with v1's
 * `xl-lock` badge. There is no start button and no "notify me" (v1's toggle never
 * persisted anything). The page h1 is the course title; the status is also stated in body
 * text, and Back to catalog is the first focusable after the heading.
 */
export default function ComingSoon({ path }: { path: Path }) {
  return (
    <div
      className="ds-card"
      style={{ padding: 26, maxWidth: 620, margin: "30px auto 0", display: "flex", flexDirection: "column", gap: 12 }}
    >
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <span
          style={{
            width: 48,
            height: 48,
            borderRadius: 12,
            background: "var(--ds-inset)",
            display: "grid",
            placeItems: "center",
            color: "var(--ds-dim)",
          }}
        >
          <Icon name={courseIcon(path.slug)} className="xl-ico--lg" />
        </span>
        <span className="xl-lock">
          <Icon name="lock" className="xl-ico--sm" /> Coming soon
        </span>
      </div>
      <div className="xl-eyebrow">Learning path · /xlearn/{path.slug}</div>
      <h1 style={{ fontSize: 24, fontWeight: 700, letterSpacing: "-.3px" }}>{path.title}</h1>
      <p style={DIM_TEXT}>{path.summary}</p>
      <div className="ds-mono" style={{ fontSize: 12, color: "var(--ds-dim)" }}>
        {path.week_total} weeks · {path.problem_total} problems
      </div>
      <p style={DIM_TEXT}>This course isn’t open yet. It opens here when it’s ready.</p>
      <div>
        <Link className="ds-btn ds-btn--secondary" to="/">
          <Icon name="grid" className="xl-ico--sm" /> Back to catalog
        </Link>
      </div>
    </div>
  );
}
