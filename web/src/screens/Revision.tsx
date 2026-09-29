import { useEffect, useMemo, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Icon } from "../components/Icon";
import { limitErrorMessage } from "../lib/api";
import { coursePath, useCourse, useCourseSlug } from "../lib/course";
import type { CourseViewBand } from "../lib/curriculum";
import {
  TOUCH_DOT_LABELS,
  bandDuration,
  bandFor,
  bandTimerSecs,
  dayLabelFor,
  formatMMSS,
  formatNextReview,
  nextReviewDate,
  scoreRevision,
  useDueRevision,
} from "../lib/revision";
import type { DueItem, RevisionProblem, ScoreInput, ScoreResult } from "../lib/revision";
import "../styles/revision.css";

// v1's re-solve timer (R-SR2), used only when the course view carries no timed band for
// the touch's level — the band's timer_s is the source (AB03).
const FALLBACK_TIMER_SECONDS = 20 * 60;
// The auto-pass pattern-naming threshold (R-SR2): named in under two minutes.
const NAME_PATTERN_MAX_SECS = 120;

const DIFF_CLASS: Record<RevisionProblem["difficulty"], string> = {
  easy: "xl-diff xl-diff--easy",
  med: "xl-diff xl-diff--med",
  hard: "xl-diff xl-diff--hard",
};
const DIFF_LABEL: Record<RevisionProblem["difficulty"], string> = { easy: "Easy", med: "Medium", hard: "Hard" };

type Bands = readonly CourseViewBand[] | undefined;

/**
 * Revision (`/xlearn/:course/revision`, AB03 v2): the course's prioritised five-touch
 * queue backed by the BFF agg (GET /paths/{course}/revision/due). Reviews are re-solves
 * from a blank editor, not re-reads, and they take priority over new problems (R-SR5).
 * Each card carries a format badge from the course manifest's revision band for its
 * touch level (label, timer or ~minutes, mock conditions). A due touch never shows the
 * pattern it tests: the gateway withholds it (m1-06), so the card shows the "Pattern
 * hidden while due" lock; the chip appears only when the payload carries the pattern —
 * a not-due, solved item, or the score result once the touch has concluded. A re-solve
 * runs on the band's timer and is auto-scored (R-SR2): a pass advances the touch, a miss
 * resets it to Day 1 (R-SR3) and opens a mistake entry.
 */
export default function Revision() {
  const course = useCourse();
  const q = useDueRevision(useCourseSlug());
  const bands = course.view?.revision?.bands;

  return (
    <div>
      <div className="xl-page-h rv-pageh">
        <div>
          <div className="xl-eyebrow">
            Spaced repetition<span className="rv-wide"> · the five-touch schedule</span>
          </div>
          <h1 className="rv-h1">Revision queue</h1>
          <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)", maxWidth: 640 }}>
            <span className="rv-wide">
              Reviews are <b style={{ color: "var(--ds-text)" }}>re-solves from a blank editor</b>, not re-reads — and
              they take priority over new problems.
            </span>
            <span className="rv-narrow">Re-solves from a blank editor, not re-reads.</span>{" "}
            {q.data && <span style={{ color: "var(--ds-text)", fontWeight: 600 }}>{q.data.dueCount} due today.</span>}
          </p>
        </div>
      </div>

      {q.isLoading && <QueueSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>{limitErrorMessage(q.error) ?? "Couldn’t load your revision queue."}</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
        </div>
      )}

      {q.data && <Queue items={q.data.items} bands={bands} />}
    </div>
  );
}

function Queue({ items, bands }: { items: DueItem[]; bands: Bands }) {
  const qc = useQueryClient();
  const slug = useCourseSlug();
  const [active, setActive] = useState<string | null>(null);
  const [result, setResult] = useState<ScoreResult | null>(null);

  const scoreM = useMutation({
    mutationFn: ({ itemId, input }: { itemId: string; input: ScoreInput }) => scoreRevision(itemId, input),
    onSuccess: (res) => setResult(res),
  });

  const dueItems = useMemo(() => items.filter((it) => it.due), [items]);
  const upcoming = useMemo(() => items.filter((it) => !it.due), [items]);
  // AB03-F6: the next review is the earliest upcoming touch the payload carries.
  const nextReview = useMemo(() => nextReviewDate(items), [items]);

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
    qc.invalidateQueries({ queryKey: ["revision", "due", slug] });
  };

  const firstDue = dueItems[0];

  const cardProps = (it: DueItem) => ({
    item: it,
    bands,
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
    <div className="rv-grid">
      <div className="rv-col">
        {dueItems.length > 0 && firstDue && active === null && (
          <div className="ds-card ds-card--teal" style={{ padding: 16, display: "flex", alignItems: "center", gap: 14, flexWrap: "wrap" }}>
            <Icon name="refresh" />
            <div className="rv-banner__txt">
              <b style={{ fontSize: 13.5 }}>{dueItems.length} review{dueItems.length === 1 ? "" : "s"} due — clear these before new problems.</b>
              <div className="rv-banner__sub" style={{ fontSize: 12, color: "var(--ds-dim)", marginTop: 2 }}>
                Start with the most fragile (Day 1) touches while they’re fresh.
              </div>
            </div>
            <button type="button" className="ds-btn ds-btn--primary ds-btn--sm rv-block" onClick={() => cardProps(firstDue).onStart()}>
              <Icon name="play" className="xl-ico--sm" /> Start next review
            </button>
          </div>
        )}

        {dueItems.length === 0 && <CaughtUp next={nextReview} slug={slug} />}

        {groups.map((g) => (
          <section key={g.level} className="xl-sect" style={{ marginBottom: 0 }}>
            <div className="xl-sect__h">
              <h2>
                Due · {dayLabelFor(g.level)}{isMockLevel(bands, g.level) ? " · mock conditions" : ""}
              </h2>
              <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-dim)" }}>
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
                <UpcomingCard key={it.itemId} item={it} bands={bands} />
              ))}
            </div>
          </section>
        )}
      </div>

      <aside className="rv-rail">
        <HowItWorks />
        <TouchLegend />
      </aside>
    </div>
  );
}

// --- AB03-F6: empty / caught up ---

function CaughtUp({ next, slug }: { next: Date | null; slug: string }) {
  return (
    <div className="ds-card ds-card--teal" role="status" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
      <Icon name="check" />
      <div className="rv-banner__txt">
        <b style={{ fontSize: 13.5 }}>
          Nothing due today.{next ? ` Next review: ${formatNextReview(next)}.` : ""}
        </b>
        <div style={{ fontSize: 12, color: "var(--ds-dim)", marginTop: 2 }}>
          New problems unlock while the queue is empty. Solve one and its five touches schedule automatically.
        </div>
      </div>
      <Link className="ds-btn ds-btn--secondary ds-btn--sm rv-block" to={coursePath(slug)}>
        Roadmap
      </Link>
    </div>
  );
}

// --- review card (collapses to a summary, expands to the re-solve panel) ---

interface CardProps {
  item: DueItem;
  bands: Bands;
  isActive: boolean;
  result: ScoreResult | null;
  submitting: boolean;
  onStart: () => void;
  onSubmit: (input: ScoreInput) => void;
  onDone: () => void;
}

function ReviewCard({ item, bands, isActive, result, submitting, onStart, onSubmit, onDone }: CardProps) {
  const band = bandFor(bands, item.touchLevel);
  const mock = isMockLevel(bands, item.touchLevel);
  // Once scored, the touch has concluded: the score response carries the pattern
  // (AB03-F3/F4), revealed in the card header and the result.
  const revealed = result?.problem?.pattern;
  return (
    <div className={isActive ? "xl-panel" : "ds-card"} style={{ padding: isActive ? 0 : 14 }}>
      <div style={{ padding: isActive ? "14px 16px" : 0, borderBottom: isActive ? "1px solid var(--ds-line)" : "none" }}>
        <ProblemLine item={item} band={band} pattern={revealed ?? item.problem?.pattern} />
        {!result && <TouchDots level={item.touchLevel} mock={mock} />}
        {!isActive && (
          <div className="rv-foot">
            <span>{touchNote(item.touchLevel, mock)}</span>
            <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm rv-block" onClick={onStart}>
              <Icon name="refresh" className="xl-ico--sm" /> Re-solve from blank
            </button>
          </div>
        )}
      </div>

      {isActive && !result && (
        <ResolvePanel item={item} timerSecs={bandTimerSecs(band) ?? FALLBACK_TIMER_SECONDS} submitting={submitting} onSubmit={onSubmit} />
      )}
      {isActive && result && <ResultPanel result={result} bands={bands} revealed={revealed} onDone={onDone} />}
    </div>
  );
}

/** One problem line (AB03-F1 row): #id, title link, difficulty, then the pattern slot,
 *  the band's format badge and v1's violet mock badge. Wide it is one wrapping row;
 *  narrow the badges drop under the title (F9, revision.css). */
function ProblemLine({ item, band, pattern }: { item: DueItem; band: CourseViewBand | undefined; pattern: string | undefined }) {
  const slug = useCourseSlug();
  const p = item.problem;
  const hasBadges = !!p || !!band || item.mockMode;
  return (
    <div className="rv-line">
      <div className="rv-line__head">
        <span className="ds-mono" style={{ fontSize: 12, color: "var(--ds-dim)" }}>#{item.problemId}</span>
        <Link
          to={coursePath(slug, "problem", item.problemId)}
          style={{ fontSize: 14.5, fontWeight: 600, color: "var(--ds-text)", textDecoration: "none" }}
        >
          {p?.title ?? `Problem ${item.problemId}`}
        </Link>
        {p && <span className={DIFF_CLASS[p.difficulty]}>{DIFF_LABEL[p.difficulty]}</span>}
      </div>
      {hasBadges && (
        <div className="rv-line__badges">
          {pattern ? (
            <PatternChip pattern={pattern} />
          ) : (
            // The payload resolved the problem but carries no pattern: the gateway withheld
            // it while the item is live (or never solved). Nothing to reveal in the DOM.
            p && (
              <span className="xl-lock">
                <Icon name="lock" className="xl-ico--sm" /> Pattern hidden while due
              </span>
            )
          )}
          {band && <FormatBadge band={band} />}
          {item.mockMode && <span className="ds-badge ds-badge--violet">mock</span>}
        </div>
      )}
    </div>
  );
}

function PatternChip({ pattern }: { pattern: string }) {
  return (
    <span className="xl-pat">
      <Icon name="layers" className="xl-ico--sm" /> {pattern}
    </span>
  );
}

/** FormatBadge is the touch format from the manifest band (AB03 rule strip): its label,
 *  "mock conditions" when the band runs in mock mode, and a timed band's mm:ss or an
 *  untimed band's "~N min". Nothing is hard-coded. */
function FormatBadge({ band }: { band: CourseViewBand }) {
  const dur = bandDuration(band);
  return (
    <span className="xl-tag rv-fmt">
      <Icon name="clock" className="xl-ico--sm" /> {band.label}
      {band.mock_mode ? " · mock conditions" : ""}
      {dur && (
        <>
          {" · "}
          <b>{dur}</b>
        </>
      )}
    </span>
  );
}

function TouchDots({ level, mock, override }: { level: number; mock?: boolean; override?: string }) {
  return (
    <>
      <div className="xl-touch" style={{ marginTop: 10 }}>
        {TOUCH_DOT_LABELS.map((_, i) => {
          const mod = i + 1 === level ? (override ?? (mock ? "xl-touch__d--mock" : "xl-touch__d--due")) : "";
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

function UpcomingCard({ item, bands }: { item: DueItem; bands: Bands }) {
  const mock = isMockLevel(bands, item.touchLevel);
  return (
    <div className="ds-card" style={{ padding: 14, opacity: 0.85 }}>
      <ProblemLine item={item} band={bandFor(bands, item.touchLevel)} pattern={item.problem?.pattern} />
      <TouchDots level={item.touchLevel} mock={mock} />
      <div className="rv-foot">
        <span>
          {dayLabelFor(item.touchLevel)}
          {mock ? " · mock conditions" : ""} · due {formatDueDate(item.dueDate)}
        </span>
        <span className="ds-badge" style={{ fontSize: 10.5 }}>Not due yet</span>
      </div>
    </div>
  );
}

// --- the re-solve panel (AB03-F2): the band's timer, a blank editor, and the three
// auto-score inputs. The pattern stays hidden: the learner names it. ---

function ResolvePanel({ item, timerSecs, submitting, onSubmit }: { item: DueItem; timerSecs: number; submitting: boolean; onSubmit: (i: ScoreInput) => void }) {
  const remaining = useCountdown(timerSecs);
  const [namedAt, setNamedAt] = useState<number | null>(null);
  const [statedComplexity, setStatedComplexity] = useState(false);
  const [solvedInTimer, setSolvedInTimer] = useState(true);
  const [draft, setDraft] = useState("");

  const expired = remaining <= 0;
  useEffect(() => {
    if (expired) setSolvedInTimer(false);
  }, [expired]);

  const elapsed = timerSecs - remaining;
  const namedSecs = namedAt ?? elapsed;
  const timerLabel = formatMMSS(timerSecs);

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
          <span style={{ fontSize: 12.5, color: "var(--ds-dim)" }}>
            <span className="rv-wide">Re-solve from blank — no peeking at the solution.</span>
            <span className="rv-narrow">No peeking at the solution.</span>
          </span>
        </span>
        <b
          className="ds-mono"
          role="timer"
          aria-label={`${formatMMSS(remaining)} remaining`}
          style={{ fontSize: 20, fontWeight: 400, fontVariantNumeric: "tabular-nums", color: expired ? "var(--ds-err)" : remaining <= 120 ? "var(--ds-warn)" : "var(--ds-teal)" }}
        >
          {formatMMSS(remaining)}
        </b>
        {/* role="timer" isn't announced; the milestones (5:00, 2:00, 0:00) are, once each. */}
        <span className="rv-sr" aria-live="polite">
          {timerMilestone(remaining, timerSecs)}
        </span>
      </div>

      <textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        aria-label="Your re-solve"
        placeholder={`// re-implement #${item.problemId} from memory — a blank editor, not a re-read`}
        spellCheck={false}
        className="ds-mono"
        style={{
          display: "block",
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

      <div className="rv-tiles">
        <ScoreCheck
          label="Named the pattern"
          hint={namedAt === null ? "tap the moment you recognise the pattern" : `named at ${formatMMSS(namedSecs)} ${namedSecs < NAME_PATTERN_MAX_SECS ? "✓ under 2 min" : "✗ over 2 min"}`}
          checked={namedAt !== null}
          ok={namedAt !== null && namedSecs < NAME_PATTERN_MAX_SECS}
          onToggle={() => setNamedAt((prev) => (prev === null ? elapsed : null))}
        />
        <ScoreCheck
          label="Solved within the timer"
          hint={expired ? `the ${timerLabel} timer elapsed` : `solved before the ${timerLabel} timer`}
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

      <p style={{ fontSize: 11.5, color: "var(--ds-dim)", margin: "0 0 12px" }}>
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
      aria-pressed={checked}
      className={`${checked ? "ds-card ds-card--interactive" : "ds-card"} rv-tile`}
      style={{ display: "flex", alignItems: "center", gap: 10, padding: "9px 12px", textAlign: "left", cursor: "pointer", width: "100%", color: "var(--ds-text)" }}
    >
      <span
        className={`xl-touch__d${checked ? (ok ? " xl-touch__d--pass" : " xl-touch__d--fail") : ""}`}
        style={{ flexShrink: 0 }}
      />
      <span style={{ flex: 1 }}>
        <span style={{ fontSize: 12.5, fontWeight: 600, display: "block" }}>{label}</span>
        <span style={{ fontSize: 11, color: "var(--ds-dim)" }}>{hint}</span>
      </span>
      {checked && <Icon name={ok ? "check" : "close"} className="xl-ico--sm" />}
    </button>
  );
}

// --- the scored result (AB03-F3 pass / F4 fail): the touch has concluded, so the
// pattern is revealed; a miss links the mistake entry it opened. ---

function ResultPanel({ result, bands, revealed, onDone }: { result: ScoreResult; bands: Bands; revealed: string | undefined; onDone: () => void }) {
  const slug = useCourseSlug();
  const pass = result.autoPass;
  const nextRef = useRef<HTMLButtonElement>(null);
  // Focus moves to "Next review" when the result lands (AB03-F3 a11y).
  useEffect(() => {
    nextRef.current?.focus();
  }, []);
  return (
    <div className="xl-panel__b" style={{ padding: 16 }}>
      <div className={`ds-card ${pass ? "ds-card--teal" : ""}`} role="status" style={{ padding: 16, borderColor: pass ? undefined : "rgba(242,109,109,.35)" }}>
        <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
          <Icon name={pass ? "check" : "alert"} className="xl-ico--sm" />
          <b style={{ fontSize: 13.5, color: pass ? "var(--ds-ok)" : "var(--ds-err)" }}>
            {pass ? `Auto-logged · Passed — advances to ${result.nextDayLabel || "the ladder’s end"}` : "Auto-logged · Missed — reset to Day 1"}
          </b>
        </div>
        <p style={{ margin: "0 0 8px", fontSize: 12.5, color: "var(--ds-dim)" }}>
          {pass
            ? result.nextTouchLevel > 0
              ? `All three auto-checks cleared. Next review ${result.nextDayLabel}${isMockLevel(bands, result.nextTouchLevel) ? " (mock conditions)" : ""}.`
              : "Day 45 cleared — this problem has completed the five-touch ladder. Strong retention."
            : "One of the three checks failed (pattern < 2 min · solved in-timer · complexity stated), so the schedule rebuilds from Day 1 — the memory needs another pass."}
        </p>
        {(revealed || !pass) && (
          <div className={`rv-reveal${pass ? " rv-reveal--patonly" : ""}`}>
            {revealed && (
              <span className="rv-reveal__pat">
                Pattern <PatternChip pattern={revealed} />
              </span>
            )}
            {revealed && !pass && (
              <span className="rv-reveal__pat" aria-hidden="true">
                ·
              </span>
            )}
            {!pass && (
              <Link to={coursePath(slug, "mistakes")} style={{ display: "inline-flex", alignItems: "center", gap: 4 }}>
                A mistake entry is open <Icon name="arrow" className="xl-ico--sm" />
              </Link>
            )}
          </div>
        )}
        <TouchDots level={pass ? Math.max(result.touchLevel, result.nextTouchLevel || result.touchLevel) : 1} override={pass ? "xl-touch__d--pass" : "xl-touch__d--fail"} />
      </div>
      <div style={{ display: "flex", gap: 8, marginTop: 14, flexWrap: "wrap" }}>
        <button ref={nextRef} type="button" className="ds-btn ds-btn--primary ds-btn--sm" onClick={onDone}>
          Next review
        </button>
        <Link className="ds-btn ds-btn--secondary ds-btn--sm" to={coursePath(slug, "problem", result.problemId)}>
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
    <span style={{ display: "inline-flex", alignItems: "center", gap: 6, fontSize: 11, color: "var(--ds-dim)" }}>
      <span className={`xl-touch__d ${mod}`} /> {label}
    </span>
  );
}

// --- helpers ---

/** isMockLevel reports whether a ladder level runs under mock conditions: the course
 *  band's mock_mode, or v1's Day 21 / Day 45 rule when the view carries no band. */
function isMockLevel(bands: Bands, level: number): boolean {
  const band = bandFor(bands, level);
  return band ? band.mock_mode : level === 4 || level === 5;
}

function touchNote(level: number, mock: boolean): string {
  if (mock) return `${dayLabelFor(level)} · mock conditions — talk aloud, no notes`;
  return `${dayLabelFor(level)} touch · re-solve from a blank editor`;
}

/** timerMilestone is the screen-reader announcement for the re-solve timer: set only at
 *  5:00, 2:00 and 0:00 (each passed once), empty before the first. */
function timerMilestone(remaining: number, total: number): string {
  if (remaining <= 0) return "Time is up.";
  if (remaining <= 120 && total > 120) return "2 minutes remaining.";
  if (remaining <= 300 && total > 300) return "5 minutes remaining.";
  return "";
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

function formatDueDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "soon";
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}

function QueueSkeleton() {
  return (
    <div className="rv-grid">
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
