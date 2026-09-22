import { Link } from "react-router-dom";
import { Icon } from "../components/Icon";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import type { Problem } from "../lib/curriculum";
import { usePathProblems } from "../lib/curriculum";
import { useProgress } from "../lib/progress";

const DIFF: Record<Problem["difficulty"], { cls: string; label: string }> = {
  easy: { cls: "xl-diff--easy", label: "EASY" },
  med: { cls: "xl-diff--med", label: "MEDIUM" },
  hard: { cls: "xl-diff--hard", label: "HARD" },
};

/**
 * Problems (`/xlearn/dsa/problems`): the practice ARENA (review round 2). A flat, by-week
 * list of every course problem — you can open + attempt ANY of them here, decoupled from
 * the curriculum timeline. Only your current week (and due revisions) counts toward the
 * course; problems ahead of it are free practice and won't count until the schedule
 * reaches them. That gate is enforced server-side; this screen just labels it.
 */
export default function Problems() {
  const problems = usePathProblems("dsa");
  const progress = useProgress();
  const frontier = progress.data?.enrolled ? progress.data.currentWeek : 0;

  return (
    <>
      <div className="xl-page-h">
        <div>
          <div className="xl-eyebrow">Data Structures &amp; Algorithms</div>
          <h1 style={{ marginTop: 6 }}>Problems</h1>
          <p>
            The practice arena — open and attempt any problem in the course. Your{" "}
            <b style={{ color: "var(--ds-text)" }}>current week</b> counts toward the course;
            anything ahead is free practice until your schedule reaches it.
          </p>
        </div>
      </div>

      {problems.isLoading && <LoadingState label="Loading problems…" />}
      {problems.isError && (
        <ErrorState message="Couldn’t load the problem list." onRetry={() => problems.refetch()} />
      )}
      {problems.data && problems.data.problems.length === 0 && (
        <EmptyState icon="list" title="No problems yet">
          Problems appear here as the curriculum is seeded.
        </EmptyState>
      )}
      {problems.data && problems.data.problems.length > 0 && (
        <ByWeek problems={problems.data.problems} frontier={frontier} />
      )}
    </>
  );
}

function ByWeek({ problems, frontier }: { problems: Problem[]; frontier: number }) {
  // Group by week, preserving seed order within a week.
  const weeks = new Map<number, Problem[]>();
  for (const p of problems) {
    const list = weeks.get(p.week_n) ?? [];
    list.push(p);
    weeks.set(p.week_n, list);
  }
  const ordered = [...weeks.keys()].sort((a, b) => a - b);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 22 }}>
      {ordered.map((n) => {
        const ahead = frontier > 0 && n > frontier;
        return (
          <div key={n}>
            <div style={{ display: "flex", alignItems: "center", gap: 10, marginBottom: 10 }}>
              <h2 style={{ fontSize: 14, fontWeight: 700 }}>Week {n}</h2>
              {frontier > 0 && n === frontier && (
                <span className="ds-badge ds-badge--ok ds-mono">Current</span>
              )}
              {ahead && (
                <span className="xl-lock">
                  <Icon name="lock" className="xl-ico--sm" /> Ahead · free practice
                </span>
              )}
            </div>
            <div className="xl-panel">
              {weeks.get(n)!.map((p, i) => (
                <ProblemRow key={p.id} problem={p} first={i === 0} />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

function ProblemRow({ problem, first }: { problem: Problem; first: boolean }) {
  const d = DIFF[problem.difficulty];
  return (
    <Link
      to={`/dsa/problem/${encodeURIComponent(problem.id)}?practice=1`}
      style={{
        display: "flex",
        alignItems: "center",
        gap: 12,
        padding: "13px 16px",
        borderTop: first ? undefined : "1px solid var(--ds-line)",
        textDecoration: "none",
        color: "inherit",
      }}
    >
      <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)", width: 34, flex: "none" }}>
        #{problem.id}
      </span>
      <b style={{ fontSize: 13.5, flex: 1, minWidth: 0 }}>{problem.title}</b>
      {d && <span className={`xl-diff ${d.cls}`}>{d.label}</span>}
      {problem.pattern && <span className="xl-pat">{problem.pattern}</span>}
      {problem.is_reinforcement && (
        <span className="ds-chip ds-chip--xs ds-mono" style={{ color: "var(--ds-muted)" }}>
          reinforce
        </span>
      )}
      <Icon name="chevron" className="xl-ico--sm" />
    </Link>
  );
}
