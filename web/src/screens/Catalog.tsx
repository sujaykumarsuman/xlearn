import { useState } from "react";
import { Link } from "react-router-dom";
import { Icon, type IconName } from "../components/Icon";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import type { Path } from "../lib/curriculum";
import { usePaths } from "../lib/curriculum";

// Per-path display icon (presentation only; the API drives which paths exist).
const PATH_ICON: Record<string, IconName> = {
  dsa: "code",
  "system-design": "server",
  "go-concurrency": "branch",
  "lld-ood": "layers",
  sql: "db",
  behavioral: "chat",
};

function pathIcon(slug: string): IconName {
  return PATH_ICON[slug] ?? "grid";
}

/** Catalog (`/xlearn`): the multi-path hub. The active DSA path renders as a hero
 *  card; every coming-soon path renders as a muted card — all from GET /paths so
 *  the list is API-driven, not hard-coded. */
export default function Catalog() {
  const paths = usePaths();

  return (
    <>
      <div style={{ marginBottom: 26 }}>
        <div className="xl-eyebrow">projects.sujaykumar.dev/xlearn</div>
        <h1 style={{ fontSize: 28, fontWeight: 700, letterSpacing: "-.4px", marginTop: 8 }}>
          Learning paths
        </h1>
        <p style={{ fontSize: 14, color: "var(--ds-dim)", marginTop: 6, maxWidth: 680 }}>
          xLearn turns a structured curriculum into a guided course that enforces the{" "}
          <b style={{ color: "var(--ds-text)" }}>method</b> — sequential unlocks, timed practice,
          five-touch spaced revision, and a mistake journal — not just a list of problems.
        </p>
      </div>

      {paths.isLoading && <LoadingState label="Loading paths…" />}

      {paths.isError && (
        <ErrorState message="Couldn’t load the catalog." onRetry={() => paths.refetch()} />
      )}

      {paths.data && <CatalogBody paths={paths.data.paths} />}
    </>
  );
}

function CatalogBody({ paths }: { paths: Path[] }) {
  const active = paths.filter((p) => p.status === "active");
  const comingSoon = paths.filter((p) => p.status === "coming_soon");

  // No paths at all — an honest empty state rather than a bare "More paths" header.
  if (active.length === 0 && comingSoon.length === 0) {
    return (
      <EmptyState icon="map" title="No learning paths yet">
        Paths appear here as they’re published. Check back soon.
      </EmptyState>
    );
  }

  return (
    <>
      {active.map((p) => (
        <ActivePathCard key={p.slug} path={p} />
      ))}

      <div className="xl-sect__h" style={{ marginBottom: 14 }}>
        <h2>More paths</h2>
        <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>
          Same method · new domains · dropping soon
        </span>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 16 }}>
        {comingSoon.map((p) => (
          <ComingSoonCard key={p.slug} path={p} />
        ))}
      </div>

      <div
        style={{
          marginTop: 24,
          padding: "16px 20px",
          background: "var(--ds-panel)",
          border: "1px solid var(--ds-line)",
          borderRadius: 12,
          display: "flex",
          alignItems: "center",
          gap: 14,
        }}
      >
        <Icon name="spark" className="" />
        <div style={{ flex: 1 }}>
          <b style={{ fontSize: 13 }}>Every path runs the same engine.</b>{" "}
          <span style={{ fontSize: 12.5, color: "var(--ds-dim)" }}>
            Guided problem stages, the five-touch revision schedule, the mistake journal, mock
            scoring, and your AI coach carry across every curriculum.
          </span>
        </div>
      </div>
    </>
  );
}

function ActivePathCard({ path }: { path: Path }) {
  return (
    <div
      className="ds-card ds-card--teal"
      style={{ padding: 22, marginBottom: 24, display: "flex", gap: 24, alignItems: "center" }}
    >
      <div
        style={{
          width: 56,
          height: 56,
          borderRadius: 14,
          background: "rgba(53,208,192,.16)",
          display: "grid",
          placeItems: "center",
          flex: "none",
        }}
      >
        <Icon name={pathIcon(path.slug)} className="xl-ico--lg" />
      </div>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 5 }}>
          <h2 style={{ fontSize: 19, fontWeight: 700 }}>{path.title}</h2>
          <span className="ds-badge ds-badge--ok ds-mono">Active</span>
        </div>
        <p style={{ fontSize: 13, color: "var(--ds-dim)", maxWidth: 560 }}>{path.summary}</p>
        <div style={{ display: "flex", alignItems: "center", gap: 18, marginTop: 14 }}>
          <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)" }}>
            {path.week_total} weeks · {path.problem_total} problems · Go-first
          </span>
        </div>
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 8, flex: "none" }}>
        <Link className="ds-btn ds-btn--primary" to="/dsa/dashboard">
          <Icon name="play" className="xl-ico--sm" /> Continue
        </Link>
        <Link className="ds-btn ds-btn--secondary ds-btn--sm" to={`/${path.slug}`}>
          View roadmap
        </Link>
      </div>
    </div>
  );
}

function ComingSoonCard({ path }: { path: Path }) {
  const [notified, setNotified] = useState(false);
  return (
    <div
      className="ds-card"
      style={{ padding: 18, display: "flex", flexDirection: "column", minHeight: 186 }}
    >
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          marginBottom: 12,
        }}
      >
        <div
          style={{
            width: 40,
            height: 40,
            borderRadius: 10,
            background: "var(--ds-inset)",
            display: "grid",
            placeItems: "center",
          }}
        >
          <Icon name={pathIcon(path.slug)} />
        </div>
        <span className="xl-lock">
          <Icon name="lock" className="xl-ico--sm" /> Coming soon
        </span>
      </div>
      <h3 style={{ fontSize: 15, fontWeight: 600, marginBottom: 5 }}>{path.title}</h3>
      <p style={{ fontSize: 12, color: "var(--ds-muted)", lineHeight: 1.5, flex: 1 }}>
        {path.summary}
      </p>
      <div
        className="ds-mono"
        style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 12, fontSize: 11, color: "var(--ds-muted)" }}
      >
        <span>
          {path.week_total} weeks · {path.problem_total} problems
        </span>
      </div>
      <button
        type="button"
        className={notified ? "ds-btn ds-btn--sm" : "ds-btn ds-btn--secondary ds-btn--sm"}
        style={{ marginTop: 12 }}
        aria-pressed={notified}
        onClick={() => setNotified((v) => !v)}
      >
        {notified ? "✓ We'll notify you" : "Notify me when it's live"}
      </button>
    </div>
  );
}

