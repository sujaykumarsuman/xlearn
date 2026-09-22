import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import type { Phase, WeekSummary } from "../lib/curriculum";
import { usePath } from "../lib/curriculum";
import { useProgress } from "../lib/progress";

const RING_R = 30;
const RING_C = 2 * Math.PI * RING_R;

const pctOf = (solved: number, total: number) => (total > 0 ? Math.round((100 * solved) / total) : 0);

/** Roadmap (`/xlearn/dsa`): the 4-phase / 16-week rail plus a summary column. Content is
 *  GET /paths/dsa; the rail's real per-user progress (overall ring, current week, streak,
 *  revisions due, phase meters) comes from GET /progress (review round 2). */
export default function Roadmap() {
  const detail = usePath("dsa");
  const progress = useProgress();
  const prog = progress.data;
  // order → completion pct, from the progress agg's by-phase rollup.
  const phasePct: Record<number, number> = {};
  for (const ph of prog?.phases ?? []) phasePct[ph.order] = pctOf(ph.solved, ph.total);

  return (
    <>
      <div className="xl-page-h">
        <div>
          <div className="xl-eyebrow">Learning path · /xlearn/dsa</div>
          <h1 style={{ marginTop: 6 }}>{detail.data?.path.title ?? "Data Structures & Algorithms"}</h1>
          <p>
            16 weeks · 4 phases · 151 problems · Go-first. The method is enforced: sequential
            unlocks, timed practice, and five-touch spaced revision.
          </p>
        </div>
        <Link className="ds-btn ds-btn--primary ds-btn--lg" to="/dsa/dashboard">
          <Icon name="play" /> Start today's plan
        </Link>
      </div>

      {detail.isLoading && <LoadingState label="Loading roadmap…" />}

      {detail.isError && (
        <ErrorState message="Couldn’t load the roadmap." onRetry={() => detail.refetch()} />
      )}

      {detail.data && detail.data.phases.length === 0 && (
        <EmptyState icon="map" title="This roadmap has no phases yet">
          Its weeks and problems will appear here once the curriculum is published.
        </EmptyState>
      )}

      {detail.data && detail.data.phases.length > 0 && (
        <div style={{ display: "grid", gridTemplateColumns: "1fr 340px", gap: 26, alignItems: "start" }}>
          <div>
            {detail.data.phases.map((phase) => (
              <PhaseBlock
                key={phase.order}
                phase={phase}
                pct={phasePct[phase.order] ?? 0}
                weeks={detail.data!.weeks.filter((w) => w.n >= phase.week_from && w.n <= phase.week_to)}
              />
            ))}
          </div>
          <SummaryColumn
            problemTotal={detail.data.path.problem_total}
            phases={detail.data.phases}
            phasePct={phasePct}
            enrolled={prog?.enrolled ?? false}
            currentWeek={prog?.currentWeek ?? 0}
            solved={prog?.summary.solved ?? 0}
            total={prog?.summary.total ?? detail.data.path.problem_total}
            streak={prog?.summary.streak.current ?? 0}
            revisionsDue={prog?.revisionsDue ?? 0}
          />
        </div>
      )}
    </>
  );
}

function PhaseBlock({ phase, weeks, pct }: { phase: Phase; weeks: WeekSummary[]; pct: number }) {
  return (
    <div style={{ marginBottom: 26 }}>
      <div style={{ display: "flex", alignItems: "baseline", gap: 10, marginBottom: 2 }}>
        <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-violet)", letterSpacing: ".5px" }}>
          PHASE {phase.order}
        </span>
        <h2 style={{ fontSize: 16, fontWeight: 700 }}>{phase.name}</h2>
        <span className="ds-chip ds-chip--xs ds-mono">
          W{phase.week_from}–{phase.week_to}
        </span>
        <span className="ds-mono" style={{ marginLeft: "auto", fontSize: 11.5, color: "var(--ds-muted)" }}>
          {pct}%
        </span>
      </div>
      <p style={{ fontSize: 12.5, color: "var(--ds-muted)", marginBottom: 14 }}>{phase.theme}</p>

      {weeks.map((w) => (
        <WeekRow key={w.n} week={w} />
      ))}
    </div>
  );
}

function WeekRow({ week }: { week: WeekSummary }) {
  return (
    <Link to={`/dsa/week/${week.n}`} style={{ display: "flex", gap: 14, textDecoration: "none", color: "inherit" }}>
      <div style={{ display: "flex", flexDirection: "column", alignItems: "center", flex: "none", width: 36 }}>
        <span
          className="ds-mono"
          style={{
            width: 36,
            height: 36,
            borderRadius: "50%",
            display: "grid",
            placeItems: "center",
            fontWeight: 600,
            fontSize: 13,
            flex: "none",
            background: "var(--ds-panel-2)",
            border: "1.6px solid var(--ds-line-2)",
            color: "var(--ds-dim)",
          }}
        >
          {week.n}
        </span>
        <span style={{ flex: 1, width: 2, background: "var(--ds-line)", minHeight: 12 }} />
      </div>
      <div className="ds-card" style={{ flex: 1, marginBottom: 10, padding: "12px 14px" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 9 }}>
          <b style={{ fontSize: 13.5, flex: 1 }}>
            Week {week.n} · {week.title}
          </b>
          {week.total > 0 && (
            <span className="ds-chip ds-chip--xs ds-mono">
              {week.total} {week.total === 1 ? "problem" : "problems"}
            </span>
          )}
        </div>
        <div style={{ marginTop: 8, fontSize: 11.5 }}>
          {week.total > 0 ? (
            <DiffMix week={week} />
          ) : (
            <span style={{ color: "var(--ds-muted)" }}>{week.thesis}</span>
          )}
        </div>
      </div>
    </Link>
  );
}

// DiffMix renders the seeded difficulty counts with the difficulty tokens:
// Easy=green (--ds-ok), Medium=amber (--ds-warn), Hard=red (--ds-err).
function DiffMix({ week }: { week: WeekSummary }) {
  const parts: ReactNode[] = [];
  if (week.easy > 0) {
    parts.push(
      <span key="e" style={{ color: "var(--ds-ok)" }}>
        {week.easy} Easy
      </span>,
    );
  }
  if (week.med > 0) {
    parts.push(
      <span key="m" style={{ color: "var(--ds-warn)" }}>
        {week.med} Med
      </span>,
    );
  }
  if (week.hard > 0) {
    parts.push(
      <span key="h" style={{ color: "var(--ds-err)" }}>
        {week.hard} Hard
      </span>,
    );
  }
  return (
    <span className="ds-mono" style={{ display: "inline-flex", gap: 8, flexWrap: "wrap" }}>
      {parts.map((p, i) => (
        <span key={i} style={{ display: "inline-flex", gap: 8 }}>
          {i > 0 && <span style={{ color: "var(--ds-muted)" }}>·</span>}
          {p}
        </span>
      ))}
    </span>
  );
}

function SummaryColumn({
  problemTotal,
  phases,
  phasePct,
  enrolled,
  currentWeek,
  solved,
  total,
  streak,
  revisionsDue,
}: {
  problemTotal: number;
  phases: Phase[];
  phasePct: Record<number, number>;
  enrolled: boolean;
  currentWeek: number;
  solved: number;
  total: number;
  streak: number;
  revisionsDue: number;
}) {
  const denom = total > 0 ? total : problemTotal;
  const pct = pctOf(solved, denom);
  const filled = (pct / 100) * RING_C;
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, position: "sticky", top: 0 }}>
      <div className="xl-panel" style={{ padding: 20, display: "flex", flexDirection: "column", alignItems: "center" }}>
        <svg width={132} height={132} viewBox="0 0 76 76" style={{ transform: "rotate(-90deg)" }}>
          <circle className="xl-ring-track" cx={38} cy={38} r={RING_R} fill="none" strokeWidth={7} />
          <circle
            cx={38}
            cy={38}
            r={RING_R}
            fill="none"
            strokeWidth={7}
            strokeLinecap="round"
            stroke="var(--ds-teal)"
            strokeDasharray={`${filled.toFixed(1)} ${RING_C.toFixed(1)}`}
          />
        </svg>
        <div style={{ marginTop: -88, textAlign: "center", marginBottom: 52 }}>
          <div className="ds-mono" style={{ fontSize: 30, fontWeight: 700 }}>
            {pct}%
          </div>
          <div style={{ fontSize: 11, color: "var(--ds-muted)" }}>
            {solved} / {denom} solved
          </div>
        </div>
        <SummaryRow
          label="Current"
          value={enrolled ? `Week ${currentWeek}` : "Not started"}
          valueColor="var(--ds-teal)"
          first
        />
        <SummaryRow label="Streak" value={enrolled ? `${streak}` : "—"} valueColor="var(--ds-warn)" />
        <SummaryRow label="Revisions due" value={enrolled ? `${revisionsDue}` : "—"} valueColor="var(--ds-dim)" />
      </div>

      <div className="xl-panel">
        <div className="xl-panel__h" style={{ padding: "12px 14px" }}>
          <Icon name="layers" className="xl-ico--sm" />
          <h3 style={{ fontSize: 13 }}>Phase progress</h3>
        </div>
        <div className="xl-panel__b" style={{ padding: 14, display: "flex", flexDirection: "column", gap: 13 }}>
          {phases.map((p) => {
            const ppct = phasePct[p.order] ?? 0;
            return (
              <div key={p.order}>
                <div style={{ display: "flex", justifyContent: "space-between", fontSize: 11.5, marginBottom: 5 }}>
                  <span style={{ color: "var(--ds-dim)" }}>{p.name}</span>
                  <span className="ds-mono xl-mut">{ppct}%</span>
                </div>
                <div className="ds-meter">
                  <div className="ds-meter__fill" style={{ width: `${ppct}%` }} />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      <Link className="ds-btn ds-btn--secondary ds-btn--block" to="/">
        Browse all paths <Icon name="arrow" className="xl-ico--sm" />
      </Link>
    </div>
  );
}

function SummaryRow({
  label,
  value,
  valueColor,
  first,
}: {
  label: string;
  value: string;
  valueColor: string;
  first?: boolean;
}) {
  return (
    <div
      style={{
        width: "100%",
        display: "flex",
        justifyContent: "space-between",
        fontSize: 12,
        color: "var(--ds-dim)",
        marginTop: first ? 0 : 8,
        borderTop: first ? "1px solid var(--ds-line)" : undefined,
        paddingTop: first ? 14 : 0,
      }}
    >
      <span>{label}</span>
      <b style={{ color: valueColor }}>{value}</b>
    </div>
  );
}
