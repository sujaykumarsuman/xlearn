import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Icon } from "../components/Icon";
import type { ConceptRef, Problem, ProblemState, Touch, WeekAggregate } from "../lib/curriculum";
import { useWeek } from "../lib/curriculum";

type Filter = "all" | "guided" | "reinf";

const DIFF_CLASS: Record<Problem["difficulty"], string> = {
  easy: "xl-diff xl-diff--easy",
  med: "xl-diff xl-diff--med",
  hard: "xl-diff xl-diff--hard",
};
const DIFF_LABEL: Record<Problem["difficulty"], string> = { easy: "Easy", med: "Medium", hard: "Hard" };

/**
 * Week (`/xlearn/dsa/week/:n`): the week thesis, its concept links, and the filtered
 * problem list, from GET /paths/dsa/weeks/:n (BFF `agg`). The five-touch dots, per-
 * problem status and the progress meter read from the aggregation's placeholder
 * `userState` — an honest empty/available state at 0/core, never faked, until
 * practice/review land (S05/S06).
 */
export default function Week() {
  const { n: nParam } = useParams();
  const n = Number(nParam);
  const validN = Number.isInteger(n) && n > 0;
  // Pass 0 for a malformed week so the hook stays disabled; the guard below renders
  // the not-found state (a bad/stale URL like /dsa/week/0 must not fall through to a
  // blank page with a "Week NaN of 16" header).
  const week = useWeek("dsa", validN ? n : 0);

  if (!validN) {
    return (
      <>
        <div className="xl-page-h">
          <div>
            <div className="xl-eyebrow">DSA · Week</div>
            <h1 style={{ marginTop: 6 }}>Week not found</h1>
          </div>
          <Link className="ds-btn ds-btn--secondary" to="/dsa">
            <Icon name="map" className="xl-ico--sm" /> Roadmap
          </Link>
        </div>
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>
            “{nParam}” isn’t a valid week — weeks are numbered 1–16.
          </span>
        </div>
      </>
    );
  }

  return (
    <>
      <WeekHeader n={n} data={week.data} />

      {week.isLoading && <WeekSkeleton />}

      {week.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>
            {week.error?.status === 404 ? `Week ${nParam} isn’t part of this path yet.` : "Couldn’t load this week."}
          </span>
          {week.error?.status !== 404 && (
            <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => week.refetch()}>
              Retry
            </button>
          )}
        </div>
      )}

      {week.data && <WeekBody n={n} data={week.data} />}
    </>
  );
}

function WeekHeader({ n, data }: { n: number; data?: WeekAggregate }) {
  const weekTotal = data?.path.week_total ?? 16;
  const phase = data?.phase;
  const eyebrow = phase
    ? `Week ${n} of ${weekTotal} · Phase ${phase.order} ${phase.name}`
    : `Week ${n} of ${weekTotal}`;
  // An honest starting suggestion (a link to a real problem), not a "resume" claim —
  // there is no per-user progress yet. Prefer the first core (non-reinforcement) one.
  const firstProblem = data && (data.problems.find((p) => !p.is_reinforcement) ?? data.problems[0]);

  return (
    <div className="xl-page-h">
      <div>
        <div className="xl-eyebrow">{eyebrow}</div>
        <h1 style={{ marginTop: 6 }}>{data?.week.title ?? `Week ${Number.isFinite(n) ? n : ""}`.trim()}</h1>
      </div>
      <div style={{ display: "flex", gap: 8 }}>
        <Link className="ds-btn ds-btn--secondary" to="/dsa">
          <Icon name="map" className="xl-ico--sm" /> Roadmap
        </Link>
        {firstProblem && (
          <Link className="ds-btn ds-btn--primary" to={`/dsa/problem/${firstProblem.id}`}>
            Start practicing <Icon name="arrow" className="xl-ico--sm" />
          </Link>
        )}
      </div>
    </div>
  );
}

function WeekBody({ n, data }: { n: number; data: WeekAggregate }) {
  return (
    <>
      <div className="ds-card ds-card--teal" style={{ padding: "18px 20px", marginBottom: 22 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 8 }}>
          <Icon name="bulb" className="xl-ico--sm" />
          <span
            className="ds-mono"
            style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", color: "var(--ds-teal)" }}
          >
            What this week is really about
          </span>
        </div>
        <p style={{ fontSize: 14, lineHeight: 1.65, color: "var(--ds-text)", maxWidth: 820 }}>{data.week.thesis}</p>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "1fr 320px", gap: 24, alignItems: "start" }}>
        <div>
          <Concepts n={n} concepts={data.concepts} />
          <Problems problems={data.problems} userState={data.userState} />
        </div>
        <RightRail n={n} data={data} />
      </div>
    </>
  );
}

function Concepts({ n, concepts }: { n: number; concepts: ConceptRef[] }) {
  return (
    <div className="xl-sect">
      <div className="xl-sect__h">
        <h2>Concepts</h2>
      </div>
      {concepts.length === 0 ? (
        <div className="xl-mut" style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>
          No concept reading for this week yet.
        </div>
      ) : (
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>
          {concepts.map((c) => (
            <Link
              key={c.slug}
              className="ds-card ds-card--interactive"
              to={`/dsa/concept/${c.slug}?week=${n}`}
              style={{ padding: 15, textDecoration: "none", color: "inherit" }}
            >
              <div style={{ display: "flex", alignItems: "center", gap: 9, marginBottom: 8 }}>
                <span
                  style={{
                    width: 30,
                    height: 30,
                    borderRadius: 8,
                    background: "rgba(155,140,240,.14)",
                    display: "grid",
                    placeItems: "center",
                  }}
                >
                  <Icon name="book" className="xl-ico--sm" />
                </span>
                <b style={{ fontSize: 14 }}>{c.title}</b>
              </div>
              <div style={{ fontSize: 11.5, color: "var(--ds-teal)", display: "flex", alignItems: "center", gap: 4 }}>
                Read concept <Icon name="arrow" className="xl-ico--sm" />
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}

const FILTERS: { key: Filter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "guided", label: "Guided" },
  { key: "reinf", label: "Reinforcement" },
];

function Problems({
  problems,
  userState,
}: {
  problems: Problem[];
  userState: WeekAggregate["userState"];
}) {
  const [filter, setFilter] = useState<Filter>("all");
  const shown = problems.filter(
    (p) => filter === "all" || (filter === "guided" && !p.is_reinforcement) || (filter === "reinf" && p.is_reinforcement),
  );

  return (
    <div className="xl-sect">
      <div className="xl-sect__h">
        <h2>Problems</h2>
        <div className="ds-seg" role="group" aria-label="Filter problems">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              className={filter === f.key ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"}
              aria-pressed={filter === f.key}
              onClick={() => setFilter(f.key)}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {problems.length === 0 ? (
        <EmptyRow label="No problems seeded for this week yet." />
      ) : shown.length === 0 ? (
        <EmptyRow label={filter === "reinf" ? "No reinforcement problems this week." : "No guided problems this week."} />
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 9 }}>
          {shown.map((p) => (
            <ProblemRow key={p.id} problem={p} state={userState.problems[p.id]} />
          ))}
        </div>
      )}
    </div>
  );
}

function ProblemRow({ problem, state }: { problem: Problem; state?: ProblemState }) {
  const locked = state?.status === "locked";
  return (
    <Link
      to={`/dsa/problem/${problem.id}`}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 14,
        padding: "13px 15px",
        background: "var(--ds-panel)",
        border: "1px solid var(--ds-line)",
        borderRadius: 10,
        textDecoration: "none",
        color: "inherit",
        opacity: locked ? 0.6 : 1,
      }}
    >
      <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)", width: 34, flex: "none" }}>
        #{problem.id}
      </span>
      <span style={{ flex: 1, minWidth: 0 }}>
        <span style={{ display: "flex", alignItems: "center", gap: 8, flexWrap: "wrap" }}>
          <b style={{ fontSize: 13.5 }}>{problem.title}</b>
          <span className={DIFF_CLASS[problem.difficulty]}>{DIFF_LABEL[problem.difficulty]}</span>
          <span className="xl-pat">{problem.pattern}</span>
          {problem.is_reinforcement && (
            <span className="ds-chip ds-chip--xs ds-mono" style={{ color: "var(--ds-violet)" }}>
              <Icon name="refresh" className="" /> reinforcement
            </span>
          )}
        </span>
      </span>
      <TouchDots touches={state?.touches} />
      <StatusChip state={state} />
      <Icon name="chevron" className="xl-ico--sm" />
    </Link>
  );
}

const TOUCH_MOD: Record<Touch["result"], string> = {
  none: "",
  pass: " xl-touch__d--pass",
  fail: " xl-touch__d--fail",
  due: " xl-touch__d--due",
  mock: " xl-touch__d--mock",
};

// Five-touch dots (Day 1/3/7/21/45). Placeholder state → all neutral; the modifier
// classes only appear once the review service reports real results (S06).
function TouchDots({ touches }: { touches?: Touch[] }) {
  const dots = touches && touches.length === 5 ? touches : Array.from({ length: 5 }, (_, i) => ({ level: i + 1, dueDate: null, result: "none" as const }));
  return (
    <span className="xl-touch" aria-label="Five-touch revision progress: not started">
      {dots.map((t) => (
        <span key={t.level} className={`xl-touch__d${TOUCH_MOD[t.result] ?? ""}`} />
      ))}
    </span>
  );
}

function StatusChip({ state }: { state?: ProblemState }) {
  const status = state?.status ?? "available";
  if (status === "locked") {
    return (
      <span className="xl-lock" style={{ width: 96, justifyContent: "center" }}>
        <Icon name="lock" className="xl-ico--sm" /> Locked
      </span>
    );
  }
  if (status === "attempting") {
    return (
      <span className="ds-badge ds-badge--info" style={{ width: 96, justifyContent: "center" }}>
        Attempting
      </span>
    );
  }
  if (status === "solved") {
    return (
      <span className="ds-badge ds-badge--ok" style={{ width: 96, justifyContent: "center" }}>
        Solved
      </span>
    );
  }
  return (
    <span className="ds-chip ds-chip--sm ds-mono" style={{ width: 96, justifyContent: "center" }}>
      Available
    </span>
  );
}

function RightRail({ n, data }: { n: number; data: WeekAggregate }) {
  const rollup = data.userState.week;
  const pct = rollup.coreTotal > 0 ? Math.round((rollup.solved / rollup.coreTotal) * 100) : 0;
  const suggestion = data.problems.find((p) => !p.is_reinforcement) ?? data.problems[0];

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
      <div className="xl-panel">
        <div className="xl-panel__b">
          <div style={{ fontSize: 12, color: "var(--ds-dim)", marginBottom: 10 }}>Week {n} progress</div>
          <div style={{ display: "flex", alignItems: "baseline", gap: 8, marginBottom: 10 }}>
            <span className="ds-mono" style={{ fontSize: 26, fontWeight: 700 }}>
              {rollup.solved}
            </span>
            <span style={{ color: "var(--ds-muted)" }}>/ {rollup.coreTotal} core solved</span>
          </div>
          <div className="ds-meter">
            <div className="ds-meter__fill" style={{ width: `${pct}%` }} />
          </div>
          <div style={{ display: "flex", gap: 14, marginTop: 14, fontSize: 12 }}>
            <span>
              <b style={{ color: "var(--ds-ok)" }}>{rollup.byDifficulty.easy}</b> <span className="xl-mut">Easy</span>
            </span>
            <span>
              <b style={{ color: "var(--ds-warn)" }}>{rollup.byDifficulty.med}</b> <span className="xl-mut">Med</span>
            </span>
            <span>
              <b style={{ color: "var(--ds-err)" }}>{rollup.byDifficulty.hard}</b> <span className="xl-mut">Hard</span>
            </span>
          </div>
        </div>
      </div>

      <div className="xl-panel">
        <div className="xl-panel__b">
          <div style={{ fontSize: 12, color: "var(--ds-dim)", marginBottom: 10 }}>Suggested for today</div>
          {suggestion ? (
            <Link
              to={`/dsa/problem/${suggestion.id}`}
              style={{
                display: "flex",
                alignItems: "center",
                gap: 10,
                padding: 10,
                borderRadius: 8,
                background: "#101a1c",
                border: "1px solid rgba(53,208,192,.3)",
                textDecoration: "none",
                color: "inherit",
              }}
            >
              <Icon name="code" className="xl-ico--sm" />
              <span style={{ flex: 1 }}>
                <b style={{ fontSize: 12.5 }}>Start #{suggestion.id} · {suggestion.title}</b>
                <br />
                <span style={{ fontSize: 11, color: "var(--ds-muted)" }}>the week’s first guided problem</span>
              </span>
              <Icon name="arrow" className="xl-ico--sm" />
            </Link>
          ) : (
            <div className="xl-mut" style={{ fontSize: 12, color: "var(--ds-muted)" }}>
              Nothing to practice here yet.
            </div>
          )}
          <div style={{ fontSize: 11.5, color: "var(--ds-muted)", marginTop: 12, lineHeight: 1.5 }}>
            Once you start solving, revisions come first — clear the due queue before new problems.
          </div>
        </div>
      </div>
    </div>
  );
}

function EmptyRow({ label }: { label: string }) {
  return (
    <div
      className="xl-mut"
      style={{
        padding: "16px 15px",
        background: "var(--ds-panel)",
        border: "1px dashed var(--ds-line-2)",
        borderRadius: 10,
        fontSize: 12.5,
        color: "var(--ds-muted)",
      }}
    >
      {label}
    </div>
  );
}

// A DS-styled skeleton for the two-column week layout while the aggregation loads.
function WeekSkeleton() {
  return (
    <>
      <div className="xl-skel" style={{ height: 74, marginBottom: 22 }} />
      <div style={{ display: "grid", gridTemplateColumns: "1fr 320px", gap: 24, alignItems: "start" }}>
        <div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginBottom: 24 }}>
            <div className="xl-skel" style={{ height: 92 }} />
            <div className="xl-skel" style={{ height: 92 }} />
          </div>
          <div style={{ display: "flex", flexDirection: "column", gap: 9 }}>
            {Array.from({ length: 5 }, (_, i) => (
              <div key={i} className="xl-skel" style={{ height: 50 }} />
            ))}
          </div>
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <div className="xl-skel" style={{ height: 150 }} />
          <div className="xl-skel" style={{ height: 120 }} />
        </div>
      </div>
    </>
  );
}
