import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { categoryLabel, useDashboard } from "../lib/mistakes";
import type { Reminder, WeakArea } from "../lib/mistakes";
import type { DueItem } from "../lib/revision";

/**
 * Dashboard / "Today" (`/xlearn/dsa/dashboard`): the S07 slice of the daily home —
 * the two surfaces this sprint owns, wired to the BFF (GET /dashboard, GET /weak-area).
 * Due reviews take priority over new problems (R-SR5), so the revisions-due panel leads;
 * the weak-area card (R-MJ3) flags the week's top mistake category. The fuller daily
 * plan, streak and progress stats land in S09.
 */
export default function Dashboard() {
  const q = useDashboard();
  const revisions = q.data?.revisions ?? null;
  const dueItems = (revisions?.items ?? []).filter((it) => it.due);
  const reminders = q.data?.reminders ?? [];
  const weakArea = q.data?.weakArea ?? null;

  return (
    <div>
      <div className="xl-page-h" style={{ borderBottom: "1px solid var(--ds-line)", paddingBottom: 16, marginBottom: 20 }}>
        <div>
          <div className="xl-eyebrow">DSA Interview Mastery</div>
          <h1 style={{ marginTop: 6, fontSize: 26, fontWeight: 700, letterSpacing: "-.3px" }}>Today</h1>
          <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)" }}>
            {todayLabel()} · reviews come before new problems.
          </p>
        </div>
        <Link className="ds-btn ds-btn--primary ds-btn--lg" to="/dsa/revision">
          <Icon name="play" /> Start reviews
        </Link>
      </div>

      {q.isLoading && <DashboardSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>Couldn’t load your dashboard.</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
        </div>
      )}

      {q.data && (
        <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 360px", gap: 22, alignItems: "start" }}>
          <RevisionsDuePanel dueItems={dueItems} reminderCount={reminders.length} />
          <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
            <WeakAreaCard weak={weakArea} />
            <RemindersCard reminders={reminders} />
          </div>
        </div>
      )}
    </div>
  );
}

function RevisionsDuePanel({ dueItems, reminderCount }: { dueItems: DueItem[]; reminderCount: number }) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h" style={{ padding: "10px 14px", display: "flex", alignItems: "center", gap: 8 }}>
        <span style={{ color: "var(--ds-teal)" }}><Icon name="refresh" /></span>
        <h3 style={{ flex: 1, fontSize: 13 }}>Revisions due today</h3>
        <span className="ds-badge ds-badge--info ds-mono">{dueItems.length}</span>
      </div>

      {dueItems.length === 0 ? (
        <div style={{ padding: 24, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="check" />
          <div>
            <b style={{ fontSize: 13.5 }}>Queue clear — no reviews due.</b>
            <div style={{ fontSize: 12, color: "var(--ds-muted)", marginTop: 2 }}>
              New problems unlock while the queue is empty.
            </div>
          </div>
          <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa" style={{ marginLeft: "auto" }}>
            Roadmap
          </Link>
        </div>
      ) : (
        <>
          <div style={{ padding: "6px 8px" }}>
            {dueItems.map((it) => (
              <DueRow key={it.itemId} item={it} />
            ))}
          </div>
          <div style={{ padding: "12px 14px", borderTop: "1px solid var(--ds-line)" }}>
            <Link className="ds-btn ds-btn--secondary ds-btn--block ds-btn--sm" to="/dsa/revision">
              Open revision queue <Icon name="arrow" className="xl-ico--sm" />
            </Link>
          </div>
        </>
      )}

      {reminderCount > 0 && (
        <div style={{ padding: "10px 14px", borderTop: "1px solid var(--ds-line)", fontSize: 11.5, color: "var(--ds-muted)", display: "flex", alignItems: "center", gap: 8 }}>
          <Icon name="bell" className="xl-ico--sm" />
          {reminderCount} in-app reminder{reminderCount === 1 ? "" : "s"} waiting.
        </div>
      )}
    </div>
  );
}

const DAY_BADGE_CLASS: Record<number, string> = {
  1: "ds-badge ds-badge--info ds-mono",
  2: "ds-badge ds-badge--info ds-mono",
  3: "ds-badge ds-badge--ok ds-mono",
  4: "ds-badge ds-badge--violet ds-mono",
  5: "ds-badge ds-badge--violet ds-mono",
};

function DueRow({ item }: { item: DueItem }) {
  const title = item.problem?.title ?? `Problem ${item.problemId}`;
  return (
    <Link
      to="/dsa/revision"
      style={{ display: "block", padding: "11px 10px", borderRadius: 8, textDecoration: "none", color: "inherit" }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)" }}>#{item.problemId}</span>
        <b style={{ fontSize: 13, flex: 1 }}>{title}</b>
        <span className={DAY_BADGE_CLASS[item.touchLevel] ?? "ds-badge ds-mono"} style={{ height: 19, fontSize: 10 }}>
          {item.dayLabel}
        </span>
      </div>
      <TouchDots level={item.touchLevel} mock={item.mockMode} />
    </Link>
  );
}

function TouchDots({ level, mock }: { level: number; mock: boolean }) {
  return (
    <div className="xl-touch" style={{ marginTop: 8 }}>
      {[1, 2, 3, 4, 5].map((l) => {
        let mod = "";
        if (l < level) mod = " xl-touch__d--pass";
        else if (l === level) mod = mock ? " xl-touch__d--mock" : " xl-touch__d--due";
        return <span key={l} className={`xl-touch__d${mod}`} />;
      })}
    </div>
  );
}

function WeakAreaCard({ weak }: { weak: WeakArea | null }) {
  if (!weak || !weak.topCategory) {
    return (
      <div className="ds-card" style={{ padding: 16 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8, color: "var(--ds-muted)" }}>
          <Icon name="alert" className="xl-ico--sm" />
          <span style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", fontWeight: 600, fontFamily: "var(--ds-font-mono)" }}>
            Weak area this week
          </span>
        </div>
        <p style={{ fontSize: 12.5, color: "var(--ds-dim)", lineHeight: 1.5, margin: 0 }}>
          No weak area yet — classify your open mistakes and one will surface here.
        </p>
        <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa/mistakes" style={{ marginTop: 12 }}>
          Open journal <Icon name="arrow" className="xl-ico--sm" />
        </Link>
      </div>
    );
  }
  const count = weak.topCount;
  return (
    <div className="ds-card ds-card--violet" style={{ padding: 16 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8, color: "var(--ds-violet)" }}>
        <Icon name="alert" className="xl-ico--sm" />
        <span style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", fontWeight: 600, fontFamily: "var(--ds-font-mono)" }}>
          Weak area this week
        </span>
      </div>
      <div style={{ fontSize: 16, fontWeight: 600, marginBottom: 4 }}>{categoryLabel(weak.topCategory)}</div>
      <p style={{ fontSize: 12.5, color: "var(--ds-dim)", lineHeight: 1.5, margin: 0 }}>
        {count} mistake {count === 1 ? "entry" : "entries"} this week. Reviews name it before you fail it.
      </p>
      <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa/mistakes" style={{ marginTop: 12 }}>
        Drill this area <Icon name="arrow" className="xl-ico--sm" />
      </Link>
    </div>
  );
}

function RemindersCard({ reminders }: { reminders: Reminder[] }) {
  if (reminders.length === 0) return null;
  return (
    <div className="xl-panel">
      <div className="xl-panel__h" style={{ padding: "10px 14px", display: "flex", alignItems: "center", gap: 8 }}>
        <Icon name="bell" className="xl-ico--sm" />
        <h3 style={{ flex: 1, fontSize: 13 }}>Reminders</h3>
        <span className="ds-badge ds-mono">{reminders.length}</span>
      </div>
      <div style={{ padding: 14, fontSize: 12, color: "var(--ds-dim)" }}>
        You have {reminders.length} in-app reminder{reminders.length === 1 ? "" : "s"} for due reviews — clear the queue to dismiss them.
      </div>
    </div>
  );
}

function todayLabel(): string {
  return new Date().toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });
}

function DashboardSkeleton() {
  return (
    <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 360px", gap: 22, alignItems: "start" }}>
      <div className="xl-skel" style={{ height: 260 }} />
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        <div className="xl-skel" style={{ height: 150 }} />
        <div className="xl-skel" style={{ height: 100 }} />
      </div>
    </div>
  );
}
