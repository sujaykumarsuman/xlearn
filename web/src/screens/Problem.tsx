import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Icon } from "../components/Icon";
import type { IconName } from "../components/Icon";
import { Markdown } from "../components/Markdown";
import type { Outcome, PracticeState, Problem as ProblemMeta, ProblemAggregate, ProblemSection, RevealPenalty } from "../lib/curriculum";
import { logOutcome, revealNext, startAttempt, useProblem } from "../lib/curriculum";

const DIFF_CLASS: Record<ProblemMeta["difficulty"], string> = {
  easy: "xl-diff xl-diff--easy",
  med: "xl-diff xl-diff--med",
  hard: "xl-diff xl-diff--hard",
};
const DIFF_LABEL: Record<ProblemMeta["difficulty"], string> = { easy: "Easy", med: "Medium", hard: "Hard" };

// Stage durations mirror the server timers (R-PF3) — used only to size the ring.
const STAGE_SECONDS: Record<"attempt" | "hint", number> = { attempt: 900, hint: 600 };

const OUTCOMES: { value: Outcome; label: string; hint: string; color: string }[] = [
  { value: "clean", label: "Clean", hint: "no help, in time", color: "var(--ds-ok)" },
  { value: "rough", label: "Rough", hint: "solved, ugly", color: "var(--ds-warn)" },
  { value: "assisted", label: "Assisted", hint: "needed a hint", color: "var(--ds-info)" },
  { value: "miss", label: "Miss", hint: "didn’t get it", color: "var(--ds-err)" },
];

const OUTCOME_LABEL: Record<Outcome, string> = { clean: "Clean", rough: "Rough", assisted: "Assisted", miss: "Miss" };

/**
 * Problem (`/xlearn/dsa/problem/:id`): the guided 3-pane workspace built on the BFF
 * agg (GET /problems/:id) — statement + only the UNLOCKED stage sections (R-PF1),
 * plus the practice state and the server-authoritative timer. The learner starts a
 * blind attempt under a 15-min timer, reveals a hint (10-min) then the solution
 * (with a reveal penalty, R-PF2), re-implements from a BLANK editor (R-PF4), and logs
 * an outcome. All stage/timer state is server-driven; the client is a mirror.
 */
export default function Problem() {
  const { id = "" } = useParams();
  const q = useProblem(id);

  return (
    <div>
      {q.isLoading && <WorkspaceSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>
            {q.error?.status === 404 ? "This problem doesn’t exist yet." : "Couldn’t load this problem."}
          </span>
          {q.error?.status !== 404 && (
            <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
              Retry
            </button>
          )}
        </div>
      )}

      {q.data && <Workspace id={id} data={q.data} />}
    </div>
  );
}

function Workspace({ id, data }: { id: string; data: ProblemAggregate }) {
  const qc = useQueryClient();
  const { problem, sections, state } = data;

  const [reimplementing, setReimplementing] = useState(false);
  const [draft, setDraft] = useState("");
  const [penalty, setPenalty] = useState<RevealPenalty | null>(null);

  const invalidate = () => qc.invalidateQueries({ queryKey: ["problem", id] });

  const startM = useMutation({ mutationFn: () => startAttempt(id), onSuccess: invalidate });
  const revealM = useMutation({
    mutationFn: () => revealNext(id),
    onSuccess: (res) => {
      setPenalty(res.penalty);
      invalidate();
    },
  });
  const outcomeM = useMutation({ mutationFn: (o: Outcome) => logOutcome(id, o), onSuccess: invalidate });

  const unlocked = new Set(state.unlockedStages);
  const solutionUnlocked = unlocked.has("solution");
  const attempting = state.status === "attempting";
  const solved = state.status === "solved";
  const notStarted = !attempting && !solved;

  return (
    <>
      <ProblemHeader problem={problem} state={state} />

      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 320px", gap: 20, alignItems: "start", marginTop: 18 }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 18, minWidth: 0 }}>
          <ReadingPane sections={sections} />
          <Editor reimplementing={reimplementing} draft={draft} setDraft={setDraft} />
        </div>

        <MethodRail
          id={id}
          state={state}
          notStarted={notStarted}
          attempting={attempting}
          solved={solved}
          solutionUnlocked={solutionUnlocked}
          reimplementing={reimplementing}
          penalty={penalty}
          onStart={() => startM.mutate()}
          starting={startM.isPending}
          onReveal={() => revealM.mutate()}
          revealing={revealM.isPending}
          onReimplement={() => setReimplementing(true)}
          onOutcome={(o) => outcomeM.mutate(o)}
          loggingOutcome={outcomeM.isPending}
        />
      </div>
    </>
  );
}

// --- header (breadcrumb + title row + timer HUD) ---

function ProblemHeader({ problem, state }: { problem: ProblemMeta; state: PracticeState }) {
  return (
    <div style={{ borderBottom: "1px solid var(--ds-line)", paddingBottom: 14 }}>
      <div className="xl-crumb" style={{ marginBottom: 10 }}>
        <Link to="/dsa" style={{ color: "inherit", textDecoration: "none" }}>
          dsa
        </Link>
        <span className="xl-crumb__sep">/</span>
        <Link to={`/dsa/week/${problem.week_n}`} style={{ color: "inherit", textDecoration: "none" }}>
          week {problem.week_n}
        </Link>
        <span className="xl-crumb__sep">/</span>
        <span style={{ color: "var(--ds-text)" }}>problem/{problem.id}</span>
      </div>

      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 16, flexWrap: "wrap" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
          <span className="ds-mono" style={{ fontSize: 13, color: "var(--ds-muted)" }}>
            #{problem.id}
          </span>
          <h1 style={{ fontSize: 24, fontWeight: 700, letterSpacing: "-.3px", margin: 0 }}>{problem.title}</h1>
          <span className={DIFF_CLASS[problem.difficulty]}>{DIFF_LABEL[problem.difficulty]}</span>
          {problem.pattern && (
            <span className="xl-pat">
              <Icon name="layers" className="xl-ico--sm" /> {problem.pattern}
            </span>
          )}
          {problem.leetcode_url && <ExtChip href={problem.leetcode_url} label="LeetCode" />}
          {problem.neetcode_url && <ExtChip href={problem.neetcode_url} label="NeetCode" />}
        </div>
        <TimerHUD state={state} />
      </div>
    </div>
  );
}

function ExtChip({ href, label }: { href: string; label: string }) {
  return (
    <a className="ds-chip ds-chip--xs" href={href} target="_blank" rel="noopener noreferrer" style={{ textDecoration: "none", color: "inherit", display: "inline-flex", alignItems: "center", gap: 4 }}>
      {label} <Icon name="ext" className="xl-ico--sm" />
    </a>
  );
}

const STAGE_TABS: { key: PracticeState["stageReached"] | "attempt"; label: string }[] = [
  { key: "attempt", label: "0 · Attempt" },
  { key: "hint", label: "1 · Hint" },
  { key: "solution", label: "2 · Solution" },
];

function TimerHUD({ state }: { state: PracticeState }) {
  const remaining = useCountdown(state.timer);
  const active = state.stageReached === "" ? "attempt" : state.stageReached;

  return (
    <div style={{ display: "flex", alignItems: "center", gap: 14 }}>
      <div className="ds-tabs" role="group" aria-label="Stage">
        {STAGE_TABS.map((tab) => (
          <span key={tab.key} className={tab.key === active ? "ds-tab ds-tab--active" : "ds-tab"}>
            {tab.label}
          </span>
        ))}
      </div>
      {state.timer ? <CountdownRing timer={state.timer} remaining={remaining} /> : <IdleClock solved={state.status === "solved"} />}
    </div>
  );
}

function IdleClock({ solved }: { solved: boolean }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 8, color: "var(--ds-muted)" }}>
      <Icon name={solved ? "check" : "clock"} className="xl-ico--sm" />
      <span className="ds-mono" style={{ fontSize: 13 }}>
        {solved ? "solved" : "untimed"}
      </span>
    </div>
  );
}

function CountdownRing({ timer, remaining }: { timer: NonNullable<PracticeState["timer"]>; remaining: number }) {
  const total = STAGE_SECONDS[timer.kind];
  const frac = total > 0 ? Math.max(0, Math.min(1, remaining / total)) : 0;
  const circ = 150.8;
  const color = remaining <= 30 ? "var(--ds-err)" : remaining <= 120 ? "var(--ds-warn)" : "var(--ds-teal)";
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
      <svg width={46} height={46} viewBox="0 0 56 56" style={{ transform: "rotate(-90deg)" }}>
        <circle className="xl-ring-track" cx={28} cy={28} r={24} fill="none" strokeWidth={4} />
        <circle cx={28} cy={28} r={24} fill="none" stroke={color} strokeWidth={4} strokeLinecap="round" strokeDasharray={`${frac * circ} ${circ}`} />
      </svg>
      <div>
        <div className="ds-mono xl-timer__t" style={{ fontSize: 18, color, fontVariantNumeric: "tabular-nums" }}>
          {formatMMSS(remaining)}
        </div>
        <div className="ds-mono" style={{ fontSize: 10, textTransform: "uppercase", letterSpacing: ".6px", color: "var(--ds-muted)" }}>
          {timer.kind}
        </div>
      </div>
    </div>
  );
}

// --- reading pane ---

const STAGE_HEADS: { stage: ProblemSection["stage"]; title: string; icon: IconName }[] = [
  { stage: "attempt", title: "Statement", icon: "list" },
  { stage: "hint", title: "Hints", icon: "key" },
  { stage: "solution", title: "Solution", icon: "check" },
];

function ReadingPane({ sections }: { sections: ProblemSection[] }) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__b" style={{ padding: 0 }}>
        {STAGE_HEADS.map(({ stage, title, icon }) => {
          const inStage = sections.filter((s) => s.stage === stage).sort((a, b) => a.order - b.order);
          if (inStage.length === 0) return null;
          return (
            <section key={stage} style={{ borderTop: "1px solid var(--ds-line)", padding: "16px 20px" }}>
              <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 10 }}>
                <Icon name={icon} className="xl-ico--sm" />
                <span className="ds-mono" style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", color: "var(--ds-teal)" }}>
                  {title}
                </span>
              </div>
              <div className="cn" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
                {inStage.map((s, i) => (
                  <SectionBlock key={`${s.stage}-${s.kind}-${i}`} section={s} />
                ))}
              </div>
            </section>
          );
        })}
      </div>
    </div>
  );
}

function SectionBlock({ section }: { section: ProblemSection }) {
  if (section.code) {
    return <pre className="xl-code">{section.code}</pre>;
  }
  return <Markdown source={section.body_md} />;
}

// --- center editor ---

const REIMPL_PLACEHOLDER = "func threeSum(nums []int) [][]int {\n    // re-implement from memory — no peeking at the solution\n\n\n}";
const STUB_TEMPLATE = "func threeSum(nums []int) [][]int {\n    sort.Ints(nums)\n    var res [][]int\n    for i := 0; i < len(nums); i++ {\n        // TODO: skip duplicate i\n        l, r := i+1, len(nums)-1\n        for l < r {\n            // TODO: move a pointer based on the sum\n        }\n    }\n    return res\n}";

function Editor({ reimplementing, draft, setDraft }: { reimplementing: boolean; draft: string; setDraft: (s: string) => void }) {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h" style={{ padding: "10px 14px", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
        <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Icon name="code" className="xl-ico--sm" />
          <span className="ds-mono" style={{ fontSize: 12 }}>
            solution.go
          </span>
        </span>
        <div className="ds-seg" role="group" aria-label="Editor language">
          <button type="button" className="ds-seg__btn ds-seg__btn--on" aria-pressed="true">
            Go
          </button>
          <button type="button" className="ds-seg__btn" disabled title="Go-first curriculum — a C++ editor comes later">
            C++
          </button>
        </div>
      </div>
      <div className="xl-panel__b" style={{ padding: 12 }}>
        {reimplementing ? (
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder={REIMPL_PLACEHOLDER}
            spellCheck={false}
            className="ds-mono"
            style={{
              width: "100%",
              minHeight: 220,
              resize: "vertical",
              background: "var(--ds-input-bg)",
              color: "var(--ds-text)",
              border: "1px solid var(--ds-line-2)",
              borderRadius: 8,
              padding: 12,
              fontSize: 12.5,
              lineHeight: 1.6,
            }}
          />
        ) : (
          <pre className="xl-code" style={{ margin: 0 }}>
            {STUB_TEMPLATE}
          </pre>
        )}
        <p style={{ fontSize: 11.5, color: "var(--ds-muted)", marginTop: 10, marginBottom: 0 }}>
          {reimplementing
            ? "Blank on purpose — re-implement from memory. v1 is self-assessed (no code execution)."
            : "Sketch your approach here. Reveal a hint or the solution from the method rail when you’re stuck."}
        </p>
      </div>
    </div>
  );
}

// --- right method rail ---

interface RailProps {
  id: string;
  state: PracticeState;
  notStarted: boolean;
  attempting: boolean;
  solved: boolean;
  solutionUnlocked: boolean;
  reimplementing: boolean;
  penalty: RevealPenalty | null;
  onStart: () => void;
  starting: boolean;
  onReveal: () => void;
  revealing: boolean;
  onReimplement: () => void;
  onOutcome: (o: Outcome) => void;
  loggingOutcome: boolean;
}

function MethodRail(p: RailProps) {
  const hintUnlocked = new Set(p.state.unlockedStages).has("hint");
  const timerRunning = !!p.state.timer && !p.state.timer.expired;

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14, position: "sticky", top: 0 }}>
      <StageStepper state={p.state} />

      {p.notStarted && (
        <div className="ds-card" style={{ padding: 16 }}>
          <p style={{ margin: "0 0 12px", fontSize: 12.5, color: "var(--ds-dim)" }}>Solve blind first — the timer starts a focused 15-minute attempt.</p>
          <button type="button" className="ds-btn ds-btn--primary ds-btn--block" onClick={p.onStart} disabled={p.starting}>
            <Icon name="play" className="xl-ico--sm" /> {p.starting ? "Starting…" : "Start attempt · 15 min"}
          </button>
        </div>
      )}

      {p.attempting && !p.solutionUnlocked && (
        <div className="ds-card" style={{ padding: 16 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
            <Icon name="lock" className="xl-ico--sm" />
            <b style={{ fontSize: 13 }}>{hintUnlocked ? "Solution locked" : "Hints & solution locked"}</b>
          </div>
          <p style={{ margin: "0 0 10px", fontSize: 12, color: "var(--ds-muted)" }}>Unlocks when the timer ends — or tap below.</p>
          {hintUnlocked && timerRunning && (
            <p style={{ display: "flex", gap: 6, margin: "0 0 10px", fontSize: 12, color: "var(--ds-warn)" }}>
              <Icon name="alert" className="xl-ico--sm" />
              <span>Revealing the full solution now means you owe #{p.id} another attempt in 3 days.</span>
            </p>
          )}
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--block" onClick={p.onReveal} disabled={p.revealing}>
            <Icon name="key" className="xl-ico--sm" /> {hintUnlocked ? "Reveal full solution" : "I’m stuck — show a hint"}
          </button>
        </div>
      )}

      {p.attempting && p.solutionUnlocked && !p.reimplementing && (
        <div className="ds-card ds-card--teal" style={{ padding: 16 }}>
          <b style={{ fontSize: 13.5 }}>Studied it? Prove it.</b>
          <p style={{ margin: "6px 0 12px", fontSize: 12.5, color: "var(--ds-dim)" }}>
            Close the solution and re-implement {`#${p.id}`} from a blank editor. This is where it actually sticks.
          </p>
          <button type="button" className="ds-btn ds-btn--primary ds-btn--block" onClick={p.onReimplement}>
            <Icon name="code" className="xl-ico--sm" /> Re-implement from memory
          </button>
        </div>
      )}

      {p.attempting && (
        <OutcomePicker onOutcome={p.onOutcome} logging={p.loggingOutcome} revealedEarly={p.state.revealedEarly} />
      )}

      {p.penalty && p.attempting && <PenaltyNote penalty={p.penalty} />}

      {p.solved && <SolvedCard state={p.state} />}
    </div>
  );
}

const STEPS: { key: "attempt" | "hint" | "solution"; label: string; hint: string }[] = [
  { key: "attempt", label: "Stage 0 · Attempt", hint: "Solve blind · 15 min" },
  { key: "hint", label: "Stage 1 · Hint", hint: "Nudge + pattern · 10 min" },
  { key: "solution", label: "Stage 2 · Solution", hint: "Full walkthrough → re-implement" },
];

function StageStepper({ state }: { state: PracticeState }) {
  const rank = (s: string) => (s === "attempt" ? 1 : s === "hint" ? 2 : s === "solution" ? 3 : 0);
  const reached = state.status === "solved" ? 3 : rank(state.stageReached);
  return (
    <div className="xl-panel">
      <div className="xl-panel__b" style={{ display: "flex", flexDirection: "column", gap: 12 }}>
        {STEPS.map((step) => {
          const r = rank(step.key);
          const mod = r < reached || (state.status === "solved") ? " xl-touch__d--pass" : r === reached ? " xl-touch__d--due" : "";
          return (
            <div key={step.key} style={{ display: "flex", alignItems: "flex-start", gap: 10 }}>
              <span className={`xl-touch__d${mod}`} style={{ marginTop: 3 }} />
              <div>
                <div style={{ fontSize: 12.5, fontWeight: 600 }}>{step.label}</div>
                <div style={{ fontSize: 11, color: "var(--ds-muted)" }}>{step.hint}</div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

function OutcomePicker({ onOutcome, logging, revealedEarly }: { onOutcome: (o: Outcome) => void; logging: boolean; revealedEarly: boolean }) {
  const [selected, setSelected] = useState<Outcome | null>(null);
  return (
    <div className="xl-panel">
      <div className="xl-panel__h" style={{ padding: "10px 14px" }}>
        <Icon name="flag" className="xl-ico--sm" />
        <h3 style={{ fontSize: 13 }}>Log your outcome</h3>
      </div>
      <div className="xl-panel__b">
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 8 }}>
          {OUTCOMES.map((o) => (
            <button
              key={o.value}
              type="button"
              className={selected === o.value ? "ds-card ds-card--teal" : "ds-card ds-card--interactive"}
              onClick={() => setSelected(o.value)}
              style={{ padding: "10px 12px", textAlign: "left", cursor: "pointer" }}
            >
              <div style={{ fontSize: 12.5, fontWeight: 700, color: o.color }}>{o.label}</div>
              <div style={{ fontSize: 10.5, color: "var(--ds-muted)" }}>{o.hint}</div>
            </button>
          ))}
        </div>
        {revealedEarly && (
          <p style={{ display: "flex", gap: 6, margin: "10px 0 0", fontSize: 11.5, color: "var(--ds-warn)" }}>
            <Icon name="alert" className="xl-ico--sm" />
            <span>You revealed the solution early — an extra re-attempt will be queued.</span>
          </p>
        )}
        <button
          type="button"
          className="ds-btn ds-btn--primary ds-btn--block"
          style={{ marginTop: 12 }}
          disabled={!selected || logging}
          onClick={() => selected && onOutcome(selected)}
        >
          {logging ? "Logging…" : "Mark solved & schedule revision"}
        </button>
      </div>
    </div>
  );
}

function PenaltyNote({ penalty }: { penalty: RevealPenalty }) {
  return (
    <div className="ds-card" style={{ padding: "12px 14px", display: "flex", gap: 8, borderColor: "rgba(240,180,41,.3)" }}>
      <Icon name="alert" className="xl-ico--sm" />
      <span style={{ fontSize: 12, color: "var(--ds-warn)" }}>{penalty.message}</span>
    </div>
  );
}

const TOUCH_LABELS = ["D1", "D3", "D7", "D21", "D45"];

function SolvedCard({ state }: { state: PracticeState }) {
  return (
    <div className="ds-card ds-card--teal" style={{ padding: 16 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
        <Icon name="check" className="xl-ico--sm" />
        <b style={{ fontSize: 13.5 }}>Logged as {state.lastOutcome ? OUTCOME_LABEL[state.lastOutcome] : "solved"}</b>
      </div>
      <p style={{ margin: "0 0 12px", fontSize: 12.5, color: "var(--ds-dim)" }}>
        Five-touch revision schedule created — reviews are re-solves from a blank editor, not re-reads.
      </p>
      <div className="xl-touch" style={{ marginBottom: 6 }}>
        {TOUCH_LABELS.map((_, i) => (
          <span key={i} className={`xl-touch__d${i === 0 ? " xl-touch__d--due" : ""}`} />
        ))}
      </div>
      <div className="xl-touch__lab" style={{ marginBottom: 12 }}>
        {TOUCH_LABELS.map((l) => (
          <span key={l}>{l}</span>
        ))}
      </div>
      {state.revealedEarly && (
        <p style={{ display: "flex", gap: 6, margin: "0 0 12px", fontSize: 11.5, color: "var(--ds-warn)" }}>
          <Icon name="alert" className="xl-ico--sm" />
          <span>+1 re-attempt queued for Day 3 — you revealed the solution early.</span>
        </p>
      )}
      <div style={{ display: "flex", gap: 8 }}>
        <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa/revision">
          Revision queue
        </Link>
        <Link className="ds-btn ds-btn--primary ds-btn--sm" to="/dsa">
          Next problem
        </Link>
      </div>
    </div>
  );
}

// --- helpers ---

// useCountdown derives a live countdown from the server timer's absolute deadline
// (R-PF3: server-authoritative). It re-seeds whenever the timer changes (a refetch
// brings a new deadline), so the client never runs a naive standalone clock.
function useCountdown(timer: PracticeState["timer"]): number {
  const [remaining, setRemaining] = useState(timer?.remainingSeconds ?? 0);
  const deadline = timer ? new Date(timer.deadlineAt).getTime() : 0;
  useEffect(() => {
    if (!timer) {
      setRemaining(0);
      return;
    }
    const tick = () => setRemaining(Math.max(0, Math.round((deadline - Date.now()) / 1000)));
    tick();
    const iv = window.setInterval(tick, 1000);
    return () => window.clearInterval(iv);
  }, [timer, deadline]);
  return remaining;
}

function formatMMSS(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

function WorkspaceSkeleton() {
  return (
    <>
      <div className="xl-skel" style={{ height: 56, marginBottom: 18 }} />
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 320px", gap: 20, alignItems: "start" }}>
        <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
          <div className="xl-skel" style={{ height: 220 }} />
          <div className="xl-skel" style={{ height: 260 }} />
        </div>
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div className="xl-skel" style={{ height: 140 }} />
          <div className="xl-skel" style={{ height: 120 }} />
        </div>
      </div>
    </>
  );
}
