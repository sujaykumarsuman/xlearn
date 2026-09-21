import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { useProgress } from "../lib/progress";
import type {
  HeatmapDay,
  PatternMastery,
  PhaseCompletion,
  ProgressData,
  ProgressSummary,
} from "../lib/progress";
import type { MockTrend } from "../lib/mock";

/**
 * Progress (`/xlearn/dsa/progress`, S09): the analytics screen, read entirely from the
 * assessment read-model projections composed with the curriculum taxonomy in the gateway
 * (GET /progress `agg`). Four stat tiles, the revision-activity heatmap, completion by
 * phase, pattern mastery, the mock rubric trend, and the first-solve outcome mix.
 */
export default function Progress() {
  const q = useProgress();

  return (
    <div>
      <div className="xl-page-h" style={{ borderBottom: "1px solid var(--ds-line)", paddingBottom: 16, marginBottom: 20 }}>
        <div>
          <div className="xl-eyebrow">Progress &amp; stats</div>
          <h1 style={{ marginTop: 6, fontSize: 26, fontWeight: 700, letterSpacing: "-.3px" }}>Your progress</h1>
          <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)" }}>
            The numbers that predict interview-readiness: coverage, retention, and mock trend.
          </p>
        </div>
      </div>

      {q.isLoading && <ProgressSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span style={{ flex: 1, color: "var(--ds-dim)" }}>Couldn’t load your progress.</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
        </div>
      )}

      {q.data && <ProgressBody data={q.data} />}
    </div>
  );
}

function ProgressBody({ data }: { data: ProgressData }) {
  return (
    <>
      <StatTiles s={data.summary} />
      <Heatmap days={data.heatmap?.days ?? []} streak={data.summary.streak} />
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 20, marginBottom: 20, alignItems: "start" }}>
        <CompletionByPhase phases={data.phases} />
        <PatternMasteryPanel patterns={data.patterns} />
      </div>
      <div style={{ display: "grid", gridTemplateColumns: "1.3fr 1fr", gap: 20, alignItems: "start" }}>
        <RubricTrend trend={data.trend} />
        <OutcomeMix mix={data.summary.outcomeMix} />
      </div>
    </>
  );
}

// --- stat tiles ---

function StatTiles({ s }: { s: ProgressSummary }) {
  const solvedPct = s.total > 0 ? Math.round((s.solved / s.total) * 100) : 0;
  const delta = s.mock.delta;
  return (
    <div style={{ display: "grid", gridTemplateColumns: "repeat(4,minmax(0,1fr))", gap: 14, marginBottom: 20 }}>
      <div className="xl-stat">
        <div className="xl-stat__l">Problems solved</div>
        <div className="xl-stat__v">
          {s.solved}
          <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / {s.total}</span>
        </div>
        <div className="ds-meter" style={{ marginTop: 10 }}>
          <div className="ds-meter__fill" style={{ width: `${solvedPct}%` }} />
        </div>
      </div>

      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-warn)" }}><Icon name="flame" className="xl-ico--sm" /></span> Current streak
        </div>
        <div className="xl-stat__v" style={{ color: "var(--ds-warn)" }}>
          {s.streak.current}
          <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> days</span>
        </div>
        <div className="xl-stat__d">Longest: {s.streak.longest} days</div>
      </div>

      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-ok)" }}><Icon name="refresh" className="xl-ico--sm" /></span> Day-7 retention
        </div>
        <div className="xl-stat__v" style={{ color: "var(--ds-ok)" }}>{s.retention.pct}%</div>
        <div className="xl-stat__d">
          {s.retention.resets === 0 ? "No resets" : `${s.retention.resets} reset${s.retention.resets === 1 ? "" : "s"}`}
        </div>
      </div>

      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-violet)" }}><Icon name="target" className="xl-ico--sm" /></span> Mock average
        </div>
        {s.mock.count === 0 ? (
          <>
            <div className="xl-stat__v">
              —<span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / 35</span>
            </div>
            <div className="xl-stat__d">No mocks yet</div>
          </>
        ) : (
          <>
            <div className="xl-stat__v">
              {s.mock.average}
              <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / 35</span>
            </div>
            <div className="xl-stat__d">
              Last: {s.mock.last}
              {delta !== 0 && (
                <span style={{ color: delta > 0 ? "var(--ds-ok)" : "var(--ds-err)" }}> {delta > 0 ? "▲" : "▼"}</span>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// --- revision-activity heatmap ---

const HEATMAP_WEEKS = 15;
const HEATMAP_RAMP = [
  "#171b23",
  "rgba(53,208,192,.28)",
  "rgba(53,208,192,.5)",
  "rgba(53,208,192,.72)",
  "var(--ds-teal)",
];

/** heatLevel maps a day's activity count to a 0..4 ramp index. */
function heatLevel(count: number): number {
  if (count <= 0) return 0;
  if (count === 1) return 1;
  if (count <= 3) return 2;
  if (count <= 5) return 3;
  return 4;
}

function isoDay(d: Date): string {
  return d.toISOString().slice(0, 10);
}

function Heatmap({ days, streak }: { days: HeatmapDay[]; streak: { current: number } }) {
  const byDate = new Map<string, number>();
  for (const d of days) byDate.set(d.date, d.solves + d.reviews);

  // Build a 15-week × 7-day grid ending today, oldest column first.
  const today = new Date();
  const total = HEATMAP_WEEKS * 7;
  const cells: { key: string; level: number; count: number }[] = [];
  for (let i = total - 1; i >= 0; i--) {
    const d = new Date(today);
    d.setUTCDate(d.getUTCDate() - i);
    const count = byDate.get(isoDay(d)) ?? 0;
    cells.push({ key: isoDay(d), level: heatLevel(count), count });
  }

  return (
    <div className="xl-panel" style={{ marginBottom: 20 }}>
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-teal)" }}><Icon name="cal" className="xl-ico--sm" /></span>
        <h3>Revision activity</h3>
        <span style={{ marginLeft: "auto", fontSize: 11.5, color: "var(--ds-muted)" }}>
          last {HEATMAP_WEEKS} weeks · reviews + solves per day
        </span>
      </div>
      <div className="xl-panel__b">
        <div style={{ display: "flex", alignItems: "flex-end", gap: 16 }}>
          <div
            style={{
              display: "grid",
              gridTemplateRows: "repeat(7,13px)",
              gridAutoFlow: "column",
              gridAutoColumns: "13px",
              gap: 3,
            }}
          >
            {cells.map((c) => (
              <span
                key={c.key}
                role="img"
                aria-label={`${c.key}: ${c.count} ${c.count === 1 ? "activity" : "activities"}`}
                title={`${c.key} · ${c.count} activit${c.count === 1 ? "y" : "ies"}`}
                style={{ width: 13, height: 13, borderRadius: 3, background: HEATMAP_RAMP[c.level] }}
              />
            ))}
          </div>
          <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 6, fontSize: 10.5, color: "var(--ds-muted)" }}>
            Less
            {HEATMAP_RAMP.map((c, i) => (
              <span key={i} style={{ width: 12, height: 12, borderRadius: 3, background: c, display: "inline-block" }} />
            ))}
            More
          </div>
        </div>
        <div style={{ marginTop: 14, display: "flex", alignItems: "center", gap: 8, fontSize: 12, color: "var(--ds-dim)" }}>
          <span style={{ color: "var(--ds-warn)" }}><Icon name="flame" className="xl-ico--sm" /></span>
          {streak.current > 0
            ? `${streak.current}-day active streak — keep clearing reviews to hold it.`
            : "No active streak yet — a solve or a review today starts one."}
        </div>
      </div>
    </div>
  );
}

// --- completion by phase ---

function CompletionByPhase({ phases }: { phases: PhaseCompletion[] }) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-violet)" }}><Icon name="layers" className="xl-ico--sm" /></span>
        <h3>Completion by phase</h3>
      </div>
      <div className="xl-panel__b" style={{ padding: 0 }}>
        <table className="xl-table">
          <thead>
            <tr>
              <th>Phase</th>
              <th>Weeks</th>
              <th style={{ textAlign: "right" }}>Solved</th>
              <th style={{ width: 150 }}>Progress</th>
            </tr>
          </thead>
          <tbody>
            {phases.length === 0 && (
              <tr>
                <td colSpan={4} style={{ color: "var(--ds-muted)", fontSize: 12.5 }}>No phase data yet.</td>
              </tr>
            )}
            {phases.map((p) => {
              const pct = p.total > 0 ? Math.round((p.solved / p.total) * 100) : 0;
              return (
                <tr key={p.order}>
                  <td><b>{p.name}</b></td>
                  <td className="ds-mono">W{p.weekFrom}–{p.weekTo}</td>
                  <td className="ds-mono" style={{ textAlign: "right", color: p.solved > 0 ? "var(--ds-text)" : undefined }}>
                    {p.solved} / {p.total}
                  </td>
                  <td>
                    <div className="ds-meter">
                      <div className="ds-meter__fill" style={{ width: `${pct}%` }} />
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// --- pattern mastery ---

/** patternColor is the mastery ramp (NOT the difficulty tokens): strong→ok, mid→teal,
 *  low→warn, none→muted. */
function patternColor(pct: number): string {
  if (pct >= 70) return "var(--ds-ok)";
  if (pct >= 45) return "var(--ds-teal)";
  if (pct >= 25) return "var(--ds-warn)";
  return "var(--ds-muted)";
}

function PatternMasteryPanel({ patterns }: { patterns: PatternMastery[] }) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-teal)" }}><Icon name="layers" className="xl-ico--sm" /></span>
        <h3>Pattern mastery</h3>
      </div>
      <div className="xl-panel__b" style={{ display: "flex", flexDirection: "column", gap: 13 }}>
        {patterns.length === 0 && (
          <div style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>Solve a few problems to build pattern mastery.</div>
        )}
        {patterns.map((p) => (
          <div key={p.name} style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <span style={{ width: 150, fontSize: 12.5, color: "var(--ds-dim)", flex: "none" }}>{p.name}</span>
            <div className="ds-meter" style={{ flex: 1 }}>
              <div className="ds-meter__fill" style={{ width: `${p.pct}%`, background: patternColor(p.pct) }} />
            </div>
            <span className="ds-mono" style={{ fontSize: 11.5, width: 56, textAlign: "right", color: "var(--ds-muted)" }}>
              {p.solved} / {p.total}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// --- mock rubric trend ---

const TREND_MIN = 7;
const TREND_MAX = 35;

/** yFor maps a /35 score to the SVG y (higher score = higher on the chart). */
function yFor(score: number): number {
  const clamped = Math.max(TREND_MIN, Math.min(TREND_MAX, score));
  return 130 - ((clamped - TREND_MIN) / (TREND_MAX - TREND_MIN)) * 110;
}

function RubricTrend({ trend }: { trend: MockTrend | null }) {
  const points = trend?.points ?? [];
  const targets = trend?.targets ?? { w13: 24, w15: 28, pre: 30 };
  const coords = points.map((p, i) => {
    const x = points.length === 1 ? 230 : 40 + (i * (420 - 40)) / (points.length - 1);
    return { x, y: yFor(p.total35), total: p.total35 };
  });
  const polyline = coords.map((c) => `${c.x},${c.y}`).join(" ");

  return (
    <div className="xl-panel">
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-violet)" }}><Icon name="target" className="xl-ico--sm" /></span>
        <h3>Mock rubric trend</h3>
        <span style={{ marginLeft: "auto" }}>
          <Link to="/dsa/mock" style={{ fontSize: 11.5 }}>View mocks →</Link>
        </span>
      </div>
      <div className="xl-panel__b">
        {points.length === 0 ? (
          <div style={{ padding: "24px 0", fontSize: 12.5, color: "var(--ds-muted)", display: "flex", alignItems: "center", gap: 10 }}>
            <Icon name="target" /> Take your first mock to start the trend vs the W13/W15/pre-interview targets.
          </div>
        ) : (
          <svg width="100%" height="150" viewBox="0 0 460 150" preserveAspectRatio="none" style={{ overflow: "visible" }}>
            <line x1="0" y1={yFor(targets.pre)} x2="460" y2={yFor(targets.pre)} stroke="rgba(87,211,154,.4)" strokeWidth="1" strokeDasharray="4 4" />
            <line x1="0" y1={yFor(targets.w15)} x2="460" y2={yFor(targets.w15)} stroke="rgba(240,180,41,.4)" strokeWidth="1" strokeDasharray="4 4" />
            <line x1="0" y1={yFor(targets.w13)} x2="460" y2={yFor(targets.w13)} stroke="rgba(155,140,240,.5)" strokeWidth="1" strokeDasharray="4 4" />
            {coords.length > 1 && (
              <polyline points={polyline} fill="none" stroke="var(--ds-teal)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />
            )}
            {coords.map((c, i) => (
              <circle key={i} cx={c.x} cy={c.y} r={i === coords.length - 1 ? 5 : 4} fill="var(--ds-teal)" stroke="#0b0d10" strokeWidth={i === coords.length - 1 ? 2 : 0} />
            ))}
          </svg>
        )}
        <div style={{ display: "flex", gap: 16, marginTop: 12, fontSize: 11 }}>
          <span style={{ color: "var(--ds-violet)" }}>— {targets.w13} by W13</span>
          <span style={{ color: "var(--ds-warn)" }}>— {targets.w15} by W15</span>
          <span style={{ color: "var(--ds-ok)" }}>— {targets.pre} pre-interview</span>
        </div>
      </div>
    </div>
  );
}

// --- outcome mix ---

function OutcomeMix({ mix }: { mix: ProgressSummary["outcomeMix"] }) {
  const total = mix.total;
  const pct = (n: number) => (total > 0 ? (n / total) * 100 : 0);
  const rows: { label: string; count: number; color: string; dot: string }[] = [
    { label: "Clean", count: mix.clean, color: "var(--ds-ok)", dot: "ds-dot ds-dot--ok" },
    { label: "Rough", count: mix.rough, color: "var(--ds-warn)", dot: "ds-dot ds-dot--progressing" },
    { label: "Assisted", count: mix.assisted, color: "var(--ds-info)", dot: "ds-dot" },
    { label: "Miss", count: mix.miss, color: "var(--ds-err)", dot: "ds-dot ds-dot--failed" },
  ];
  return (
    <div className="xl-panel">
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-teal)" }}><Icon name="flag" className="xl-ico--sm" /></span>
        <h3>Outcome mix</h3>
        <span style={{ marginLeft: "auto", fontSize: 11, color: "var(--ds-muted)" }}>{total} first-solves</span>
      </div>
      <div className="xl-panel__b">
        {total === 0 ? (
          <div style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>Log your first solve to see the outcome mix.</div>
        ) : (
          <div style={{ display: "flex", height: 14, borderRadius: 7, overflow: "hidden", marginBottom: 16 }}>
            {rows.map((r) => r.count > 0 && <div key={r.label} style={{ width: `${pct(r.count)}%`, background: r.color }} />)}
          </div>
        )}
        <div style={{ display: "flex", flexDirection: "column", gap: 10, fontSize: 12.5 }}>
          {rows.map((r) => (
            <div key={r.label} style={{ display: "flex", alignItems: "center", gap: 8 }}>
              <span className={r.dot} style={r.label === "Assisted" ? { background: "var(--ds-info)" } : undefined} />
              <span style={{ flex: 1, color: "var(--ds-dim)" }}>{r.label}</span>
              <b className="ds-mono">{r.count}</b>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// --- skeleton ---

function ProgressSkeleton() {
  return (
    <div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4,minmax(0,1fr))", gap: 14, marginBottom: 20 }}>
        {[0, 1, 2, 3].map((i) => (
          <div key={i} className="xl-skel" style={{ height: 92 }} />
        ))}
      </div>
      <div className="xl-skel" style={{ height: 150, marginBottom: 20 }} />
      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 20 }}>
        <div className="xl-skel" style={{ height: 220 }} />
        <div className="xl-skel" style={{ height: 220 }} />
      </div>
    </div>
  );
}
