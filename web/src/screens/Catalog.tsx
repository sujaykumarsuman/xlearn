import { useState } from "react";
import { Link } from "react-router-dom";
import { Icon, type IconName } from "../components/Icon";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { useMe, type Me } from "../lib/auth";
import type { Path } from "../lib/curriculum";
import { usePaths } from "../lib/curriculum";
import { useDashboard } from "../lib/dashboard";
import { currentDay, enrollmentFor, useStartPath } from "../lib/enrollment";

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

/** Catalog (`/xlearn`): the sidebar-less home — a bare list of paths to pick (F001).
 *  The active path is enrollment-aware (F002): "Start path" until started, then a
 *  Day N · streak · today summary. Every coming-soon path renders as a muted card. All
 *  from GET /paths + GET /me so the list is API-driven, not hard-coded. */
export default function Catalog() {
  const paths = usePaths();
  const me = useMe();

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

      {paths.data && <CatalogBody paths={paths.data.paths} me={me.data} />}
    </>
  );
}

function CatalogBody({ paths, me }: { paths: Path[]; me: Me | undefined }) {
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
        <ActivePathCard key={p.slug} path={p} me={me} />
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

/** The active path's hero card. Not started → a Start CTA; started → a live summary
 *  (current day, streak, what's scheduled today) with Continue. */
function ActivePathCard({ path, me }: { path: Path; me: Me | undefined }) {
  const enrollment = enrollmentFor(me, path.slug);
  const started = enrollment !== null;
  const startPath = useStartPath();
  // The Day/streak/today summary reuses the Dashboard agg — fetched only once started
  // (and only for the real DSA path the agg is pinned to).
  const dash = useDashboard(started && path.slug === "dsa");
  const day = currentDay(enrollment);

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
          {started ? (
            <span className="ds-badge ds-badge--ok ds-mono">Day {day}</span>
          ) : (
            <span className="ds-badge ds-mono">Not started</span>
          )}
        </div>
        <p style={{ fontSize: 13, color: "var(--ds-dim)", maxWidth: 560 }}>{path.summary}</p>

        {started ? (
          <StartedSummary
            day={day}
            streak={dash.data?.stats.streak.current ?? null}
            revisionsDue={dash.data?.stats.revisionsDue ?? null}
            loading={dash.isLoading}
          />
        ) : (
          <div style={{ display: "flex", alignItems: "center", gap: 18, marginTop: 14 }}>
            <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)" }}>
              {path.week_total} weeks · {path.problem_total} problems · Go-first
            </span>
          </div>
        )}
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 8, flex: "none" }}>
        {started ? (
          <Link className="ds-btn ds-btn--primary" to={`/${path.slug}/dashboard`}>
            <Icon name="play" className="xl-ico--sm" /> Continue
          </Link>
        ) : (
          <button
            type="button"
            className="ds-btn ds-btn--primary"
            disabled={startPath.isPending}
            onClick={() => startPath.mutate(path.slug)}
          >
            <Icon name="play" className="xl-ico--sm" />{" "}
            {startPath.isPending ? "Starting…" : "Start path"}
          </button>
        )}
        <Link className="ds-btn ds-btn--secondary ds-btn--sm" to={`/${path.slug}`}>
          View roadmap
        </Link>
        {startPath.isError && (
          <span role="alert" style={{ fontSize: 11, color: "var(--ds-err)", maxWidth: 150 }}>
            Couldn’t start — try again.
          </span>
        )}
      </div>
    </div>
  );
}

/** The started path's inline summary: current day, active streak, and what's scheduled
 *  today (reviews lead). Degrades to a calm line while the agg loads or when empty. */
function StartedSummary({
  day,
  streak,
  revisionsDue,
  loading,
}: {
  day: number | null;
  streak: number | null;
  revisionsDue: number | null;
  loading: boolean;
}) {
  const today =
    revisionsDue != null && revisionsDue > 0
      ? `${revisionsDue} revision${revisionsDue === 1 ? "" : "s"} due`
      : loading
        ? "…"
        : "New problems ready";

  return (
    <div
      style={{ display: "flex", alignItems: "center", gap: 22, marginTop: 14, flexWrap: "wrap" }}
      className="ds-mono"
    >
      <SummaryStat icon="today" label="Current day" value={day != null ? `Day ${day}` : "—"} />
      <SummaryStat
        icon="flame"
        color="var(--ds-warn)"
        label="Active streak"
        value={streak != null ? `${streak}-day` : loading ? "…" : "0-day"}
      />
      <SummaryStat icon="refresh" label="Scheduled today" value={today} />
    </div>
  );
}

function SummaryStat({
  icon,
  label,
  value,
  color = "var(--ds-teal)",
}: {
  icon: IconName;
  label: string;
  value: string;
  color?: string;
}) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
      <Icon name={icon} className="xl-ico--sm" style={{ color }} />
      <span style={{ fontSize: 12 }}>
        <span style={{ color: "var(--ds-muted)" }}>{label}:</span>{" "}
        <b style={{ color: "var(--ds-text)" }}>{value}</b>
      </span>
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
