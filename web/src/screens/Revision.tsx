import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Icon } from "../components/Icon";
import {
  TOUCH_DOT_LABELS,
  dayLabelFor,
  scoreRevision,
  useDueRevision,
} from "../lib/revision";
import type { DueItem, RevisionProblem, ScoreInput, ScoreResult } from "../lib/revision";

// The re-solve runs on a 20-minute timer (R-SR2) — a re-solve from a blank editor,
// never a re-read.
const REVISION_TIMER_SECONDS = 20 * 60;
// The auto-pass pattern-naming threshold (R-SR2): named in under two minutes.
const NAME_PATTERN_MAX_SECS = 120;

const DIFF_CLASS: Record<RevisionProblem["difficulty"], string> = {
  easy: "xl-diff xl-diff--easy",
  med: "xl-diff xl-diff--med",
  hard: "xl-diff xl-diff--hard",
};
const DIFF_LABEL: Record<RevisionProblem["difficulty"], string> = { easy: "Easy", med: "Medium", hard: "Hard" };

/**
 * Revision (`/xlearn/dsa/revision`): the prioritised five-touch queue backed by the BFF
 * agg (GET /revision/due). Reviews are re-solves from a blank editor, not re-reads, and
 * they take priority over new problems (R-SR5). Each re-solve runs on a 20:00 timer and
 * is auto-scored (R-SR2): a pass advances the touch (Day 1→3→7→21→45), a miss resets it
 * to Day 1 (R-SR3). Day 21 & 45 run under mock conditions (R-SR4).
 */
export default function Revision() {
  const q = useDueRevision();

  return (
    <div>
      <div className="xl-page-h" style={{ borderBottom: "1px solid var(--ds-line)", paddingBottom: 16, marginBottom: 20 }}>
        <div>
          <div className="xl-eyebrow">Spaced repetition · the five-touch schedule</div>
          <h1 style={{ marginTop: 6, fontSize: 26, fontWeight: 700, letterSpacing: "-.3px" }}>Revision queue</h1>
          <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)", maxWidth: 640 }}>
            Reviews are <b style={{ color: "var(--ds-text)" }}>re-solves from a blank editor</b>, not re-reads — and they
            take priority over new problems.{" "}
            {q.data && <span style={{ color: "var(--ds-text)", fontWeight: 600 }}>{q.data.dueCount} due today.</span>}
          </p>
        </div>
      </div>

      {q.isLoading && <QueueSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>Couldn’t load your revision queue.</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
        </div>
      )}

      {q.data && <Queue items={q.data.items} />}
    </div>
  );
}

function Queue({ items }: { items: DueItem[] }) {
  const qc = useQueryClient();
  const [active, setActive] = useState<string | null>(null);
  const [result, setResult] = useState<ScoreResult | null>(null);

  const scoreM = useMutation({
    mutationFn: ({ itemId, input }: { itemId: string; input: ScoreInput }) => scoreRevision(itemId, input),
    onSuccess: (res) => setResult(res),
  });

  const dueItems = useMemo(() => items.filter((it) => it.due), [items]);
  const upcoming = useMemo(() => items.filter((it) => !it.due), [items]);

  // Group the due items by touch day (Day 1 → Day 45), in ladder order.
  const groups = useMemo(() => {
    const byLevel = new Map<number, DueItem[]>();
    for (const it of dueItems) {
      const g = byLevel.get(it.touchLevel) ?? [];
      g.push(it);
      byLevel.set(it.touchLevel, g);
    }
    return [1, 2, 3, 4, 5]
      .filter((lvl) => byLevel.has(lvl))
      .map((lvl) => ({ level: lvl, items: byLevel.get(lvl)! }));
  }, [dueItems]);

  const closeReview = () => {
    setActive(null);
    setResult(null);
    scoreM.reset();
    qc.invalidateQueries({ queryKey: ["revision", "due"] });
  };

  const firstDue = dueItems[0];

  const cardProps = (it: DueItem) => ({
    item: it,
    isActive: active === it.itemId,
    result: active === it.itemId ? result : null,
    submitting: scoreM.isPending,
    onStart: () => {
      setResult(null);
      scoreM.reset();
      setActive(it.itemId);
    },
    onSubmit: (input: ScoreInput) => scoreM.mutate({ itemId: it.itemId, input }),
    onDone: closeReview,
  });

  return (
    <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 300px", gap: 22, alignItems: "start" }}>
      <div style={{ display: "flex", flexDirection: "column", gap: 22, minWidth: 0 }}>
        {dueItems.length > 0 && firstDue && active === null && (
          <div className="ds-card ds-card--teal" style={{ padding: 16, display: "flex", alignItems: "center", gap: 14, flexWrap: "wrap" }}>
            <Icon name="refresh" />
            <div style={{ flex: 1, minWidth: 200 }}>
              <b style={{ fontSize: 13.5 }}>{dueItems.length} review{dueItems.length === 1 ? "" : "s"} due — clear these before new problems.</b>
              <div style={{ fontSize: 12, color: "var(--ds-dim)", marginTop: 2 }}>
                Start with the most fragile (Day 1) touches while they’re fresh.
              </div>
            </div>
            <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" onClick={() => cardProps(firstDue).onStart()}>
              <Icon name="play" className="xl-ico--sm" /> Start next review
            </button>
          </div>
        )}

        {dueItems.length === 0 && (
          <div className="ds-card ds-card--teal" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
            <Icon name="check" />
            <div>
              <b style={{ fontSize: 13.5 }}>Queue clear — no reviews due.</b>
              <div style={{ fontSize: 12, color: "var(--ds-dim)", marginTop: 2 }}>
                New problems unlock while the queue is empty. Solve one and its five touches schedule automatically.
              </div>
            </div>
            <Link className="ds-btn ds-btn--secondary ds-btn--sm" to="/dsa" style={{ marginLeft: "auto" }}>
              Roadmap
            </Link>
          </div>
        )}

        {groups.map((g) => (
          <section key={g.level} className="xl-sect" style={{ marginBottom: 0 }}>
            <div className="xl-sect__h">
              <h2>
                Due · {dayLabelFor(g.level)}{isMockLevel(g.level) ? " · mock conditions" : ""}
              </h2>
              <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-muted)" }}>
                touch {g.level}
              </span>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {g.items.map((it) => (
                <ReviewCard key={it.itemId} {...cardProps(it)} />
              ))}
            </div>
          </section>
        ))}

        {upcoming.length > 0 && (
          <section className="xl-sect" style={{ marginBottom: 0 }}>
            <div className="xl-sect__h">
              <h2>Coming up</h2>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              {upcoming.map((it) => (
                <UpcomingCard key={it.itemId} item={it} />
              ))}
            </div>
          </section>
        )}
      </div>

      <aside style={{ display: "flex", flexDirection: "column", gap: 16, position: "sticky", top: 0 }}>
        <HowItWorks />
        <TouchLegend />
      </aside>
    </div>
  );
}

// --- review card (collapses to a summary, expands to the re-solve panel) ---

interface CardProps {
  item: DueItem;
  isActive: boolean;
  result: ScoreResult | null;
  submitting: boolean;
  onStart: () => void;
  onSubmit: (input: ScoreInput) => void;
  onDone: () => void;
}

function ReviewCard({ item, isActive, result, submitting, onStart, onSubmit, onDone }: CardProps) {
  return (
    <div className={isActive ? "xl-panel" : "ds-card"} style={{ padding: isActive ? 0 : 14 }}>
      <div style={{ padding: isActive ? "14px 16px" : 0, borderBottom: isActive ? "1px solid var(--ds-line)" : "none" }}>
        <ProblemLine item={item} />
        <TouchDots level={item.touchLevel} />
        {!isActive && (
          <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 10, marginTop: 12 }}>
            <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>{touchNote(item.touchLevel)}</span>
            <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={onStart}>
              <Icon name="refresh" className="xl-ico--sm" /> Re-solve from blank
            </button>
          </div>
        )}
      </div>

      {isActive && !result && <ResolvePanel item={item} submitting={submitting} onSubmit={onSubmit} />}
      {isActive && result && <ResultPanel result={result} onDone={onDone} />}
    </div>
  );
}

function ProblemLine({ item }: { item: DueItem }) {
  const p = item.problem;
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10, flexWrap: "wrap" }}>
      <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-muted)" }}>#{item.problemId}</span>
      <Link
        to={`/dsa/problem/${encodeURIComponent(item.problemId)}`}
        style={{ fontSize: 14.5, fontWeight: 600, color: "var(--ds-text)", textDecoration: "none" }}
      >
        {p?.title ?? `Problem ${item.problemId}`}
      </Link>
      {p && <span className={DIFF_CLASS[p.difficulty]}>{DIFF_LABEL[p.difficulty]}</span>}
      {p?.pattern && (
        <span className="xl-pat">
          <Icon name="layers" className="xl-ico--sm" /> {p.pattern}
        </span>
      )}
      {item.mockMode && <span className="ds-badge ds-badge--violet">mock</span>}
    </div>
  );
}

function TouchDots({ level, override }: { level: number; override?: string }) {
  return (
    <>
      <div className="xl-touch" style={{ marginTop: 10 }}>
        {TOUCH_DOT_LABELS.map((_, i) => {
          const mod = i + 1 === level ? (override ?? (isMockLevel(level) ? "xl-touch__d--mock" : "xl-touch__d--due")) : "";
          return <span key={i} className={`xl-touch__d${mod ? " " + mod : ""}`} />;
        })}
      </div>
      <div className="xl-touch__lab" style={{ marginTop: 4 }}>
        {TOUCH_DOT_LABELS.map((l) => (
          <span key={l}>{l}</span>
        ))}
      </div>
    </>
  );
}

function UpcomingCard({ item }: { item: DueItem }) {
  return (
    <div className="ds-card" style={{ padding: 14, opacity: 0.85 }}>
      <ProblemLine item={item} />
      <TouchDots level={item.touchLevel} />
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 10, marginTop: 12 }}>
        <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>
          {dayLabelFor(item.touchLevel)}
          {isMockLevel(item.touchLevel) ? " · mock conditions" : ""} · due {formatDueDate(item.dueDate)}
        </span>
        <span className="ds-badge" style={{ fontSize: 10.5 }}>Not due yet</span>
      </div>
    </div>
  );
}

// --- the re-solve panel: a 20:00 timer, a blank editor, and the three auto-score inputs ---

function ResolvePanel({ item, submitting, onSubmit }: { item: DueItem; submitting: boolean; onSubmit: (i: ScoreInput) => void }) {
  const remaining = useCountdown(REVISION_TIMER_SECONDS);
  const [namedAt, setNamedAt] = useState<number | null>(null);
  const [statedComplexity, setStatedComplexity] = useState(false);
  const [solvedInTimer, setSolvedInTimer] = useState(true);
  const [draft, setDraft] = useState("");

  const expired = remaining <= 0;
  useEffect(() => {
    if (expired) setSolvedInTimer(false);
  }, [expired]);

  const elapsed = REVISION_TIMER_SECONDS - remaining;
  const namedSecs = namedAt ?? elapsed;

  // If the learner never tapped "Named the pattern", submit a value that FAILS the
  // R-SR2 pattern check (>= 2 min) rather than the small elapsed time — an un-affirmed
  // pattern must not auto-pass the criterion. This matches the tile's own ok logic.
  const submit = () =>
    onSubmit({ namedPatternSecs: namedAt ?? NAME_PATTERN_MAX_SECS, solvedInTimer, statedComplexity });

  return (
    <div className="xl-panel__b" style={{ padding: 16 }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12, marginBottom: 12, flexWrap: "wrap" }}>
        <span style={{ display: "flex", alignItems: "center", gap: 8 }}>
          <Icon name="clock" className="xl-ico--sm" />
          <span style={{ fontSize: 12.5, color: "var(--ds-dim)" }}>Re-solve from blank — no peeking at the solution.</span>
        </span>
        <span
          className="ds-mono"
          style={{ fontSize: 20, fontVariantNumeric: "tabular-nums", color: expired ? "var(--ds-err)" : remaining <= 120 ? "var(--ds-warn)" : "var(--ds-teal)" }}
        >
          {formatMMSS(remaining)}
        </span>
      </div>

      <textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        placeholder={`// re-implement #${item.problemId} from memory — a blank editor, not a re-read`}
        spellCheck={false}
        className="ds-mono"
        style={{
          width: "100%",
          minHeight: 150,
          resize: "vertical",
          background: "var(--ds-input-bg)",
          color: "var(--ds-text)",
          border: "1px solid var(--ds-line-2)",
          borderRadius: 8,
          padding: 12,
          fontSize: 12.5,
          lineHeight: 1.6,
          marginBottom: 12,
        }}
      />

      <div style={{ display: "flex", flexDirection: "column", gap: 8, marginBottom: 12 }}>
        <ScoreCheck
          label="Named the pattern"
          hint={namedAt === null ? "tap the moment you recognise the pattern" : `named at ${formatMMSS(namedSecs)} ${namedSecs < NAME_PATTERN_MAX_SECS ? "✓ under 2 min" : "✗ over 2 min"}`}
          checked={namedAt !== null}
          ok={namedAt !== null && namedSecs < NAME_PATTERN_MAX_SECS}
          onToggle={() => setNamedAt((prev) => (prev === null ? elapsed : null))}
        />
        <ScoreCheck
          label="Solved within the timer"
          hint={expired ? "the 20:00 timer elapsed" : "solved before the 20:00 timer"}
          checked={solvedInTimer}
          ok={solvedInTimer}
          onToggle={() => setSolvedInTimer((v) => !v)}
        />
        <ScoreCheck
          label="Stated the complexity"
          hint="time & space, out loud"
          checked={statedComplexity}
          ok={statedComplexity}
          onToggle={() => setStatedComplexity((v) => !v)}
        />
      </div>

      <p style={{ fontSize: 11.5, color: "var(--ds-muted)", margin: "0 0 12px" }}>
        Scored automatically — passes only if the pattern is named in under 2 minutes, solved in-timer, and complexity
        stated. Any miss resets this problem to Day 1.
      </p>

      <button type="button" className="ds-btn ds-btn--primary ds-btn--block" onClick={submit} disabled={submitting}>
        {submitting ? "Scoring…" : "Submit re-solve for auto-score"}
      </button>
    </div>
  );
}

function ScoreCheck({ label, hint, checked, ok, onToggle }: { label: string; hint: string; checked: boolean; ok: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className={checked ? "ds-card ds-card--interactive" : "ds-card"}
      style={{ display: "flex", alignItems: "center", gap: 10, padding: "9px 12px", textAlign: "left", cursor: "pointer", width: "100%" }}
    >
      <span
        className={`xl-touch__d${checked ? (ok ? " xl-touch__d--pass" : " xl-touch__d--fail") : ""}`}
        style={{ flexShrink: 0 }}
      />
      <span style={{ flex: 1 }}>
        <span style={{ fontSize: 12.5, fontWeight: 600, display: "block" }}>{label}</span>
        <span style={{ fontSize: 10.5, color: "var(--ds-muted)" }}>{hint}</span>
      </span>
      {checked && <Icon name={ok ? "check" : "close"} className="xl-ico--sm" />}
    </button>
  );
}

function ResultPanel({ result, onDone }: { result: ScoreResult; onDone: () => void }) {
  const pass = result.autoPass;
  return (
    <div className="xl-panel__b" style={{ padding: 16 }}>
      <div className={`ds-card ${pass ? "ds-card--teal" : ""}`} style={{ padding: 16, borderColor: pass ? undefined : "rgba(230,90,90,.35)" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
          <Icon name={pass ? "check" : "alert"} className="xl-ico--sm" />
          <b style={{ fontSize: 13.5, color: pass ? "var(--ds-ok)" : "var(--ds-err)" }}>
            {pass ? `Auto-logged · Passed — advances to ${result.nextDayLabel || "the ladder’s end"}` : "Auto-logged · Missed — reset to Day 1"}
          </b>
        </div>
        <p style={{ margin: "0 0 6px", fontSize: 12.5, color: "var(--ds-dim)" }}>
          {pass
            ? result.nextTouchLevel > 0
              ? `All three auto-checks cleared. Next review ${result.nextDayLabel}${isMockLevel(result.nextTouchLevel) ? " (mock conditions)" : ""}.`
              : "Day 45 cleared — this problem has completed the five-touch ladder. Strong retention."
            : "One of the three checks failed (pattern < 2 min · solved in-timer · complexity stated), so the schedule rebuilds from Day 1 — the memory needs another pass."}
        </p>
        <TouchDots level={pass ? Math.max(result.touchLevel, result.nextTouchLevel || result.touchLevel) : 1} override={pass ? "xl-touch__d--pass" : "xl-touch__d--fail"} />
      </div>
      <div style={{ display: "flex", gap: 8, marginTop: 14 }}>
        <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" onClick={onDone}>
          Next review
        </button>
        <Link className="ds-btn ds-btn--secondary ds-btn--sm" to={`/dsa/problem/${encodeURIComponent(result.problemId)}`}>
          Open #{result.problemId}
        </Link>
      </div>
    </div>
  );
}

// --- right rail ---

const HOW_STEPS = [
  "Re-solve from a blank editor with a 20-min timer. Never a re-read.",
  "Auto-scored — passes only if: pattern named < 2 min · solved in-timer · complexity stated.",
  "A miss on any check resets the clock to Day 1.",
  "Day 21 & 45 run under mock conditions — talk aloud, no notes.",
];

function HowItWorks() {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h" style={{ padding: "10px 14px", display: "flex", alignItems: "center", gap: 8 }}>
        <Icon name="bulb" className="xl-ico--sm" />
        <h3 style={{ fontSize: 13 }}>How a review works</h3>
      </div>
      <div className="xl-panel__b" style={{ padding: 14, display: "flex", flexDirection: "column", gap: 10 }}>
        {HOW_STEPS.map((step, i) => (
          <div key={i} style={{ display: "flex", gap: 10, alignItems: "flex-start" }}>
            <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-teal)", fontWeight: 700, marginTop: 1 }}>
              0{i + 1}
            </span>
            <span style={{ fontSize: 12, color: "var(--ds-dim)", lineHeight: 1.5 }}>{step}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function TouchLegend() {
  return (
    <div className="xl-panel">
      <div className="xl-panel__h" style={{ padding: "10px 14px", display: "flex", alignItems: "center", gap: 8 }}>
        <Icon name="refresh" className="xl-ico--sm" />
        <h3 style={{ fontSize: 13 }}>The five touches</h3>
      </div>
      <div className="xl-panel__b" style={{ padding: 14 }}>
        <div className="xl-touch">
          {TOUCH_DOT_LABELS.map((_, i) => (
            <span key={i} className="xl-touch__d" />
          ))}
        </div>
        <div className="xl-touch__lab" style={{ marginTop: 4, marginBottom: 12 }}>
          {[1, 3, 7, 21, 45].map((d) => (
            <span key={d}>{d}</span>
          ))}
        </div>
        <div style={{ display: "flex", gap: 14, flexWrap: "wrap" }}>
          <LegendKey mod="xl-touch__d--pass" label="passed" />
          <LegendKey mod="xl-touch__d--due" label="due" />
          <LegendKey mod="xl-touch__d--fail" label="failed" />
        </div>
      </div>
    </div>
  );
}

function LegendKey({ mod, label }: { mod: string; label: string }) {
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 6, fontSize: 11, color: "var(--ds-muted)" }}>
      <span className={`xl-touch__d ${mod}`} /> {label}
    </span>
  );
}

// --- helpers ---

function isMockLevel(level: number): boolean {
  return level === 4 || level === 5;
}

function touchNote(level: number): string {
  if (isMockLevel(level)) return `${dayLabelFor(level)} · mock conditions — talk aloud, no notes`;
  return `${dayLabelFor(level)} touch · re-solve from a blank editor`;
}

// useCountdown counts down from `seconds` once per second, clamped at 0. It re-seeds
// only when the starting value changes (a fresh re-solve), not on every render.
function useCountdown(seconds: number): number {
  const [remaining, setRemaining] = useState(seconds);
  useEffect(() => {
    setRemaining(seconds);
    const deadline = Date.now() + seconds * 1000;
    const tick = () => setRemaining(Math.max(0, Math.round((deadline - Date.now()) / 1000)));
    tick();
    const iv = window.setInterval(tick, 1000);
    return () => window.clearInterval(iv);
  }, [seconds]);
  return remaining;
}

function formatMMSS(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

function formatDueDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "soon";
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function QueueSkeleton() {
  return (
    <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 300px", gap: 22, alignItems: "start" }}>
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <div className="xl-skel" style={{ height: 64 }} />
        <div className="xl-skel" style={{ height: 96 }} />
        <div className="xl-skel" style={{ height: 96 }} />
      </div>
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <div className="xl-skel" style={{ height: 160 }} />
        <div className="xl-skel" style={{ height: 120 }} />
      </div>
    </div>
  );
}
