// Shared, presentational Progress views (F009): the revision-activity heatmap, the
// completion-by-phase table, and the pattern-mastery bars. Extracted from the Progress
// screen so the PUBLIC user dashboard (/xlearn/u/<username>) renders the SAME visuals from
// the same source — one implementation, no drift. These are pure/props-only (no data
// fetching), so both the authed Progress screen and the public profile feed them data.
import { Icon } from "./Icon";
import type { HeatmapDay, PatternMastery, PhaseCompletion } from "../lib/progress";

// --- revision-activity heatmap ---

export const HEATMAP_WEEKS = 15;
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

/**
 * HeatmapGrid is the bare N-week × 7-day activity grid (+ optional legend). It is the reusable
 * core of the heatmap so the full-width Progress panel and the compact public-profile card
 * render the SAME visual from one implementation. `legend`: "side" (grid + legend in a row,
 * the Progress panel), "below" (stacked, the narrow profile column) or "none".
 */
export function HeatmapGrid({
  days,
  weeks = HEATMAP_WEEKS,
  cell = 13,
  legend = "side",
}: {
  days: HeatmapDay[];
  weeks?: number;
  cell?: number;
  legend?: "side" | "below" | "none";
}) {
  const byDate = new Map<string, number>();
  for (const d of days) byDate.set(d.date, d.solves + d.reviews);

  // Build a weeks × 7-day grid ending today, oldest column first.
  const today = new Date();
  const total = weeks * 7;
  const cells: { key: string; level: number; count: number }[] = [];
  for (let i = total - 1; i >= 0; i--) {
    const d = new Date(today);
    d.setUTCDate(d.getUTCDate() - i);
    const count = byDate.get(isoDay(d)) ?? 0;
    cells.push({ key: isoDay(d), level: heatLevel(count), count });
  }

  const grid = (
    <div
      style={{
        display: "grid",
        gridTemplateRows: `repeat(7,${cell}px)`,
        gridAutoFlow: "column",
        gridAutoColumns: `${cell}px`,
        gap: 3,
      }}
    >
      {cells.map((c) => (
        <span
          key={c.key}
          role="img"
          aria-label={`${c.key}: ${c.count} ${c.count === 1 ? "activity" : "activities"}`}
          title={`${c.key} · ${c.count} activit${c.count === 1 ? "y" : "ies"}`}
          style={{ width: cell, height: cell, borderRadius: 3, background: HEATMAP_RAMP[c.level] }}
        />
      ))}
    </div>
  );

  if (legend === "none") return grid;

  const legendEl = (
    <div style={{ display: "flex", alignItems: "center", gap: 6, fontSize: 10.5, color: "var(--ds-muted)" }}>
      Less
      {HEATMAP_RAMP.map((c, i) => (
        <span key={i} style={{ width: 12, height: 12, borderRadius: 3, background: c, display: "inline-block" }} />
      ))}
      More
    </div>
  );

  if (legend === "below") {
    return (
      <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
        {grid}
        {legendEl}
      </div>
    );
  }
  return (
    <div style={{ display: "flex", alignItems: "flex-end", gap: 16 }}>
      {grid}
      <div style={{ marginLeft: "auto" }}>{legendEl}</div>
    </div>
  );
}

/** Heatmap is the full-width "Revision activity" panel (the Progress screen). */
export function Heatmap({ days, streak, weeks = HEATMAP_WEEKS }: { days: HeatmapDay[]; streak: { current: number }; weeks?: number }) {
  return (
    <div className="xl-panel" style={{ marginBottom: 20 }}>
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-teal)" }}><Icon name="cal" className="xl-ico--sm" /></span>
        <h3>Revision activity</h3>
        <span style={{ marginLeft: "auto", fontSize: 11.5, color: "var(--ds-muted)" }}>
          last {weeks} weeks · reviews + solves per day
        </span>
      </div>
      <div className="xl-panel__b">
        <HeatmapGrid days={days} weeks={weeks} legend="side" />
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

export function CompletionByPhase({ phases }: { phases: PhaseCompletion[] }) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h">
        <span style={{ color: "var(--ds-violet)" }}><Icon name="layers" className="xl-ico--sm" /></span>
        <h3>Completion by phase</h3>
        <span style={{ marginLeft: "auto" }}><OutcomeLegend /></span>
      </div>
      <div className="xl-panel__b" style={{ padding: 0 }}>
        <table className="xl-table">
          <thead>
            <tr>
              <th>Phase</th>
              <th>Weeks</th>
              <th>Progress</th>
            </tr>
          </thead>
          <tbody>
            {phases.length === 0 && (
              <tr>
                <td colSpan={3} style={{ color: "var(--ds-muted)", fontSize: 12.5 }}>No phase data yet.</td>
              </tr>
            )}
            {phases.map((p) => (
              <tr key={p.order}>
                <td><b>{p.name}</b></td>
                <td className="ds-mono">W{p.weekFrom}–{p.weekTo}</td>
                <td>
                  <OutcomeBar p={p} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

/** The segment colours for a first-solve outcome (F009 review): clean=green, rough=amber,
 *  assisted=blue, miss=red — the unsolved remainder is the track background. */
const OUTCOME_SEGMENTS = [
  { key: "clean", label: "Clean", color: "var(--ds-ok)" },
  { key: "rough", label: "Rough", color: "var(--ds-warn)" },
  { key: "assisted", label: "Assisted", color: "var(--ds-info)" },
  { key: "miss", label: "Miss", color: "var(--ds-err)" },
] as const;

/** OutcomeBar is the segmented completion meter: the solved problems coloured by outcome,
 *  the rest left as the empty track. The solved/total count + full breakdown surface on
 *  hover (title) and to assistive tech (aria-label), so no wrapping number column is needed. */
function OutcomeBar({ p }: { p: PhaseCompletion }) {
  const unsolved = Math.max(0, p.total - p.solved);
  const summary =
    p.total === 0
      ? "No problems yet"
      : `${p.solved} / ${p.total} solved · ${p.clean} clean · ${p.rough} rough · ${p.assisted} assisted · ${p.miss} miss · ${unsolved} unsolved`;
  return (
    <div className="ds-meter" style={{ display: "flex" }} role="img" title={summary} aria-label={summary}>
      {p.total > 0 &&
        OUTCOME_SEGMENTS.map((s) => {
          const n = p[s.key];
          if (!n) return null;
          return <span key={s.key} style={{ width: `${(n / p.total) * 100}%`, background: s.color, flexShrink: 0 }} />;
        })}
    </div>
  );
}

/** A compact colour key for the outcome segments, shown in the panel header. */
function OutcomeLegend() {
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 10, fontSize: 10.5, color: "var(--ds-muted)" }}>
      {OUTCOME_SEGMENTS.map((s) => (
        <span key={s.key} style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
          <span style={{ width: 8, height: 8, borderRadius: 2, background: s.color, display: "inline-block" }} />
          {s.label}
        </span>
      ))}
    </span>
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

export function PatternMasteryPanel({ patterns }: { patterns: PatternMastery[] }) {
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
