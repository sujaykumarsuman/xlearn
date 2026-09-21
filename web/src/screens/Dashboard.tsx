import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { useDashboard } from "../lib/dashboard";
import type { DashboardData, DashboardWeek, PlanItem, PlanProblem, PlanReview } from "../lib/dashboard";
import { categoryLabel } from "../lib/mistakes";
import type { Reminder, WeakArea } from "../lib/mistakes";
import type { DueItem } from "../lib/revision";

/**
 * Dashboard / "Today" (`/xlearn/dsa/dashboard`, S09): the daily home, composed by the
 * gateway from the assessment projections (streak / solved / mock), review (due queue,
 * weak-area, reminders) and curriculum (the week/problem taxonomy). Reviews come before
 * new work (R-SR5): the plan leads with due revisions, then the week's next problems.
 */
export default function Dashboard() {
  const q = useDashboard();
  const data = q.data;

  return (
    <div>
      <div className="xl-page-h" style={{ borderBottom: "1px solid var(--ds-line)", paddingBottom: 16, marginBottom: 20 }}>
        <div>
          <div className="xl-eyebrow">DSA Interview Mastery</div>
          <h1 style={{ marginTop: 6, fontSize: 26, fontWeight: 700, letterSpacing: "-.3px" }}>Today</h1>
          <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)" }}>
            {todayLabel()}
            {data?.week ? ` · Week ${data.week.n}${data.week.title ? ` — ${data.week.title}` : ""}` : ""} · reviews come before new problems.
          </p>
        </div>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
          {data && data.stats.streak.current > 0 && (
            <div className="xl-streak" title={`${data.stats.streak.current}-day streak`} style={{ fontSize: 15 }}>
              <span style={{ color: "var(--ds-warn)" }}><Icon name="flame" /></span> {data.stats.streak.current}
            </div>
          )}
          <Link className="ds-btn ds-btn--primary ds-btn--lg" to={startTarget(data)}>
            <Icon name="play" /> {data && data.stats.revisionsDue > 0 ? "Start reviews" : "Start today’s plan"}
          </Link>
        </div>
      </div>

      {q.isLoading && <DashboardSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span style={{ flex: 1, color: "var(--ds-dim)" }}>Couldn’t load your dashboard.</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
        </div>
      )}

      {data && (
        <>
          <QuickStats data={data} />
          <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 396px", gap: 22, alignItems: "start" }}>
            <div className="xl-sect" style={{ margin: 0 }}>
              <TodaysPlan plan={data.plan} />
              {data.week && <WeekProgress week={data.week} />}
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
              <RevisionsDuePanel dueItems={(data.revisions?.items ?? []).filter((it) => it.due)} reminderCount={data.reminders.length} />
              <WeakAreaCard weak={data.weakArea} />
              <RemindersCard reminders={data.reminders} />
            </div>
          </div>
        </>
      )}
    </div>
  );
}

// --- quick stats ---

function QuickStats({ data }: { data: DashboardData }) {
  const { stats } = data;
  const solvedPct = stats.solved.total > 0 ? Math.round((stats.solved.count / stats.solved.total) * 100) : 0;
  const dueBreakdown = dueByDay(data.revisions?.items ?? []);
  return (
    <div style={{ display: "grid", gridTemplateColumns: "repeat(4,minmax(0,1fr))", gap: 14, marginBottom: 24 }}>
      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-warn)" }}><Icon name="flame" className="xl-ico--sm" /></span> Streak
        </div>
        <div className="xl-stat__v" style={{ color: "var(--ds-warn)" }}>
          {stats.streak.current}
          <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> days</span>
        </div>
        <div className="xl-stat__d">Personal best: {stats.streak.longest}</div>
      </div>

      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-teal)" }}><Icon name="check" className="xl-ico--sm" /></span> Solved
        </div>
        <div className="xl-stat__v">
          {stats.solved.count}
          <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / {stats.solved.total}</span>
        </div>
        <div className="ds-meter" style={{ marginTop: 10 }}>
          <div className="ds-meter__fill" style={{ width: `${solvedPct}%` }} />
        </div>
      </div>

      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-teal)" }}><Icon name="refresh" className="xl-ico--sm" /></span> Revisions due
        </div>
        <div className="xl-stat__v" style={{ color: "var(--ds-teal)" }}>{stats.revisionsDue}</div>
        <div className="xl-stat__d">{dueBreakdown || "Queue clear"}</div>
      </div>

      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-violet)" }}><Icon name="target" className="xl-ico--sm" /></span> Mock best
        </div>
        {stats.mock.count === 0 ? (
          <>
            <div className="xl-stat__v">—<span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / 35</span></div>
            <div className="xl-stat__d">No mocks yet</div>
          </>
        ) : (
          <>
            <div className="xl-stat__v">
              {stats.mock.best}
              <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / 35</span>
            </div>
            <div className="xl-stat__d">
              Target by W13: <b style={{ color: "var(--ds-dim)" }}>{stats.mock.target}</b>{" "}
              {stats.mock.best >= stats.mock.target ? "✓" : ""}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// --- today's plan ---

function TodaysPlan({ plan }: { plan: PlanItem[] }) {
  const done = 0; // completion tracking lands with per-item state; the plan is the queue.
  return (
    <>
      <div className="xl-sect__h">
        <h2>Today’s plan</h2>
        <span className="ds-chip ds-chip--sm ds-mono">{done} / {plan.length} done</span>
      </div>
      {plan.length === 0 ? (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="check" />
          <div style={{ flex: 1 }}>
            <b style={{ fontSize: 13.5 }}>All caught up.</b>
            <div style={{ fontSize: 12, color: "var(--ds-muted)", marginTop: 2 }}>No reviews due and this week’s core problems are solved.</div>
          </div>
          <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa">Roadmap</Link>
        </div>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
          {plan.map((item, i) =>
            item.kind === "review" ? <ReviewCard key={`r-${item.itemId}`} item={item} /> : <ProblemCard key={`p-${item.problemId}`} item={item} first={i === 0} />,
          )}
        </div>
      )}
    </>
  );
}

function ReviewCard({ item }: { item: PlanReview }) {
  return (
    <Link
      to="/dsa/revision"
      style={{ display: "flex", gap: 14, alignItems: "center", padding: "13px 15px", background: "var(--ds-panel)", border: "1px solid var(--ds-line)", borderRadius: 10, textDecoration: "none", color: "inherit" }}
    >
      <span style={{ width: 28, height: 28, borderRadius: 8, background: "rgba(53,208,192,.14)", display: "grid", placeItems: "center", flex: "none" }}>
        <span style={{ color: "var(--ds-teal)" }}><Icon name="refresh" className="xl-ico--sm" /></span>
      </span>
      <span style={{ flex: 1 }}>
        <span style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
          <b style={{ fontSize: 13.5 }}>Review · re-solve <span className="ds-mono">#{item.problemId}</span> {item.title}</b>
          <span className="ds-badge ds-badge--info ds-mono" style={{ height: 19, fontSize: 10 }}>{item.dayLabel}</span>
          {item.mockMode && <span className="ds-badge ds-badge--violet ds-mono" style={{ height: 19, fontSize: 10 }}>Mock</span>}
        </span>
        <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>From a blank editor · 20-min timer</span>
      </span>
      <span className="ds-mono" style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>20 min</span>
    </Link>
  );
}

function ProblemCard({ item, first }: { item: PlanProblem; first: boolean }) {
  const active = item.status === "attempting";
  const border = first || active ? "1px solid rgba(53,208,192,.5)" : "1px solid var(--ds-line)";
  const bg = first || active ? "#101a1c" : "var(--ds-panel)";
  return (
    <Link
      to={`/dsa/problem/${item.problemId}`}
      style={{ display: "flex", gap: 14, alignItems: "center", padding: 15, background: bg, border, borderRadius: 10, textDecoration: "none", color: "inherit" }}
    >
      <span style={{ width: 28, height: 28, borderRadius: 8, background: "rgba(53,208,192,.16)", display: "grid", placeItems: "center", flex: "none" }}>
        <span style={{ color: "var(--ds-teal)" }}><Icon name="code" className="xl-ico--sm" /></span>
      </span>
      <span style={{ flex: 1 }}>
        <span style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
          <b style={{ fontSize: 14 }}>{active ? "Primary problem" : "New problem"} · <span className="ds-mono">#{item.problemId}</span> {item.title}</b>
          <span className={diffClass(item.difficulty)}>{diffLabel(item.difficulty)}</span>
          {item.pattern && <span className="xl-pat">{item.pattern}</span>}
        </span>
        <span style={{ fontSize: 12, color: "var(--ds-dim)" }}>Guided flow · 15-min attempt, then gated hints</span>
      </span>
      <span style={{ display: "flex", alignItems: "center", gap: 12 }}>
        {active && <span className="ds-mono" style={{ fontSize: 11.5, color: "var(--ds-teal)" }}>in progress</span>}
        <span className="ds-btn ds-btn--primary ds-btn--sm">{active ? "Resume" : "Start"}</span>
      </span>
    </Link>
  );
}

// --- week progress ---

function WeekProgress({ week }: { week: DashboardWeek }) {
  return (
    <div className="xl-panel" style={{ marginTop: 18 }}>
      <div className="xl-panel__b" style={{ padding: "14px 16px" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 10 }}>
          <span style={{ fontSize: 12.5, color: "var(--ds-dim)" }}>Week {week.n} progress</span>
          <Link to={`/dsa/week/${week.n}`} style={{ fontSize: 12, display: "inline-flex", alignItems: "center", gap: 4 }}>
            Open week <Icon name="arrow" className="xl-ico--sm" />
          </Link>
        </div>
        <div style={{ display: "flex", gap: 6 }}>
          {week.problems.map((p) => (
            <div key={p.problemId} title={p.title} style={{ flex: 1, height: 6, borderRadius: 3, background: segColor(p.status) }} />
          ))}
        </div>
        <div style={{ marginTop: 8, fontSize: 11.5, color: "var(--ds-muted)" }}>
          {week.solved} of {week.total} core problems solved
        </div>
      </div>
    </div>
  );
}

// --- right column: revisions / weak area / reminders ---

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
            <div style={{ fontSize: 12, color: "var(--ds-muted)", marginTop: 2 }}>New problems unlock while the queue is empty.</div>
          </div>
          <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa" style={{ marginLeft: "auto" }}>Roadmap</Link>
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
    <Link to="/dsa/revision" style={{ display: "block", padding: "11px 10px", borderRadius: 8, textDecoration: "none", color: "inherit" }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)" }}>#{item.problemId}</span>
        <b style={{ fontSize: 13, flex: 1 }}>{title}</b>
        <span className={DAY_BADGE_CLASS[item.touchLevel] ?? "ds-badge ds-mono"} style={{ height: 19, fontSize: 10 }}>{item.dayLabel}</span>
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
          <span style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", fontWeight: 600, fontFamily: "var(--ds-font-mono)" }}>Weak area this week</span>
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
        <span style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", fontWeight: 600, fontFamily: "var(--ds-font-mono)" }}>Weak area this week</span>
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

// --- helpers ---

function diffClass(d: string): string {
  return `xl-diff xl-diff--${d === "med" ? "med" : d === "hard" ? "hard" : "easy"}`;
}
function diffLabel(d: string): string {
  return d === "med" ? "Medium" : d === "hard" ? "Hard" : "Easy";
}
function segColor(status: string): string {
  if (status === "solved") return "var(--ds-ok)";
  if (status === "attempting") return "var(--ds-teal)";
  return "#1b2028";
}

function dueByDay(items: DueItem[]): string {
  const counts = new Map<string, number>();
  for (const it of items) {
    if (!it.due) continue;
    counts.set(it.dayLabel, (counts.get(it.dayLabel) ?? 0) + 1);
  }
  return [...counts.entries()].map(([label, n]) => `${n} · ${label}`).join("  •  ");
}

function startTarget(data: DashboardData | undefined): string {
  if (!data) return "/dsa/revision";
  if (data.stats.revisionsDue > 0) return "/dsa/revision";
  const firstProblem = data.plan.find((p): p is PlanProblem => p.kind === "problem");
  return firstProblem ? `/dsa/problem/${firstProblem.problemId}` : "/dsa";
}

function todayLabel(): string {
  return new Date().toLocaleDateString(undefined, { weekday: "long", month: "long", day: "numeric" });
}

function DashboardSkeleton() {
  return (
    <div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4,minmax(0,1fr))", gap: 14, marginBottom: 24 }}>
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="xl-skel" style={{ height: 92 }} />
        ))}
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 396px", gap: 22, alignItems: "start" }}>
        <div className="xl-skel" style={{ height: 300 }} />
        <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
          <div className="xl-skel" style={{ height: 200 }} />
          <div className="xl-skel" style={{ height: 120 }} />
        </div>
      </div>
    </div>
  );
}
