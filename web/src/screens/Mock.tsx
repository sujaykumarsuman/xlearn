import { useEffect, useMemo, useState } from "react";
import type { CSSProperties, ReactNode } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Icon } from "../components/Icon";
import {
  MOCK_TOTAL_SECS,
  RUBRIC_DIMENSIONS,
  scoreMock,
  startMock,
  useMock,
  useMockTrend,
} from "../lib/mock";
import type { Difficulty, MockPhase, MockSession, MockTrend, RubricScores } from "../lib/mock";

// Setup presets (problem set + difficulty), mapped to the API's setId/problemId. Set 07
// pins a concrete problem (the gateway enriches its title/pattern); "Mixed" leaves the
// problem to the server (empty id -> the header falls back to the set name).
const PROBLEM_SETS = [
  { id: "set-07", title: "Set 07 · Two Pointers", note: "1 Medium · matches Week 2", problemId: "16" },
  { id: "set-08", title: "Set 08 · Mixed", note: "random from solved", problemId: "" },
] as const;

const DIFFS: { key: Difficulty; label: string }[] = [
  { key: "easy", label: "Easy" },
  { key: "med", label: "Medium" },
  { key: "hard", label: "Hard" },
];

/**
 * Mock (`/xlearn/dsa/mock`): the timed mock interview. Setup -> a 45-minute,
 * server-authoritative live session whose phase rail (R-MK1) walks the six interview
 * phases -> the 7-dimension rubric (R-MK2, /35) with a radar + a trend chart vs the
 * readiness targets (R-MK3). The 45-min timer and phase index come from the server
 * (GET /mocks/{id}), polled while live; the client HUD only mirrors them.
 */
export default function Mock() {
  const qc = useQueryClient();
  const [mockId, setMockId] = useState("");
  const q = useMock(mockId);
  const session = q.data;

  const startM = useMutation({
    mutationFn: startMock,
    onSuccess: (s) => {
      qc.setQueryData(["mock", s.id], s);
      setMockId(s.id);
    },
  });

  const reset = () => {
    setMockId("");
    startM.reset();
  };

  if (mockId === "") {
    return <Setup onStart={(setup) => startM.mutate(setup)} starting={startM.isPending} failed={startM.isError} />;
  }

  if (q.isError) {
    return (
      <div>
        <PageHeader eyebrow="45-minute timed session" title="Mock interview" />
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>Couldn’t load this mock session.</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={reset}>
            New mock
          </button>
        </div>
      </div>
    );
  }

  if (!session) {
    return <MockSkeleton />;
  }

  if (session.status === "scored") {
    return <Results session={session} onNewMock={reset} />;
  }
  return <Live session={session} onNewMock={reset} />;
}

// --- setup ---

function Setup({
  onStart,
  starting,
  failed,
}: {
  onStart: (setup: { setId: string; problemId: string; difficulty: Difficulty }) => void;
  starting: boolean;
  failed: boolean;
}) {
  const [setIdx, setSetIdx] = useState(0);
  const [difficulty, setDifficulty] = useState<Difficulty>("med");
  const [talkAloud, setTalkAloud] = useState(true);
  const chosen = PROBLEM_SETS[setIdx] ?? PROBLEM_SETS[0];

  const start = () => onStart({ setId: chosen.id, problemId: chosen.problemId, difficulty });

  return (
    <div>
      <PageHeader
        eyebrow="45-minute timed session"
        title="Mock interview"
        sub="One problem, real interview conditions: talk aloud, no notes, phase-timed. Scored on 7 dimensions."
      />
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 380px", gap: 22, alignItems: "start" }}>
        <div className="xl-panel">
          <div className="xl-panel__h" style={{ padding: "10px 14px", display: "flex", alignItems: "center", gap: 8 }}>
            <Icon name="target" className="xl-ico--sm" />
            <h3 style={{ fontSize: 13 }}>Set up your mock</h3>
          </div>
          <div className="xl-panel__b" style={{ padding: 16, display: "flex", flexDirection: "column", gap: 16 }}>
            <div className="ds-field">
              <label className="ds-field__label">Problem set</label>
              <div style={{ display: "flex", gap: 8 }}>
                {PROBLEM_SETS.map((s, i) => (
                  <button
                    key={s.id}
                    type="button"
                    className={i === setIdx ? "ds-card ds-card--teal" : "ds-card"}
                    onClick={() => setSetIdx(i)}
                    aria-pressed={i === setIdx}
                    style={{ flex: 1, padding: 12, textAlign: "left", cursor: "pointer", color: "var(--ds-text)" }}
                  >
                    <b style={{ fontSize: 13 }}>{s.title}</b>
                    <br />
                    <span style={{ fontSize: 11, color: "var(--ds-muted)" }}>{s.note}</span>
                  </button>
                ))}
              </div>
            </div>

            <div className="ds-field">
              <label className="ds-field__label">Difficulty</label>
              <div className="ds-seg">
                {DIFFS.map((d) => (
                  <button
                    key={d.key}
                    type="button"
                    className={`ds-seg__btn${difficulty === d.key ? " ds-seg__btn--on" : ""}`}
                    aria-pressed={difficulty === d.key}
                    onClick={() => setDifficulty(d.key)}
                  >
                    {d.label}
                  </button>
                ))}
              </div>
            </div>

            <label className={`ds-toggle${talkAloud ? " ds-toggle--on" : ""}`}>
              <input
                type="checkbox"
                checked={talkAloud}
                onChange={(e) => setTalkAloud(e.target.checked)}
                style={{ position: "absolute", opacity: 0, width: 0, height: 0 }}
              />
              <span className="ds-toggle__track">
                <span className="ds-toggle__knob" />
              </span>
              Talk-aloud reminders (recommended)
            </label>

            {failed && <div className="ds-field__error">Couldn’t start the mock. Try again.</div>}

            <button type="button" className="ds-btn ds-btn--primary ds-btn--lg" onClick={start} disabled={starting}>
              <Icon name="play" /> {starting ? "Starting…" : "Start 45-minute mock"}
            </button>
          </div>
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <div className="xl-panel">
            <div className="xl-panel__b" style={{ padding: 16 }}>
              <div className="ds-mono" style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".5px", color: "var(--ds-muted)", marginBottom: 12 }}>
                The 45-minute rail
              </div>
              <div style={{ display: "flex", flexDirection: "column", gap: 9, fontSize: 12.5 }}>
                {RAIL_REFERENCE.map((r) => (
                  <div key={r.range} style={{ display: "flex", gap: 10 }}>
                    <span className="ds-mono" style={{ color: "var(--ds-teal)", width: 46, flex: "none" }}>{r.range}</span>
                    <span style={{ color: "var(--ds-dim)" }}>{r.label}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
          <div className="ds-card ds-card--violet" style={{ padding: "14px 16px" }}>
            <div className="ds-mono" style={{ fontSize: 12, color: "var(--ds-violet)", marginBottom: 6 }}>SCORED ON 7 DIMENSIONS</div>
            <div style={{ fontSize: 12, color: "var(--ds-dim)", lineHeight: 1.6 }}>
              {RUBRIC_DIMENSIONS.map((d) => d.name).join(" · ")} — each 1–5, total /35.
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

const RAIL_REFERENCE = [
  { range: "0–5", label: "Clarify the problem" },
  { range: "5–10", label: "Brute force out loud" },
  { range: "10–18", label: "Key observation → plan" },
  { range: "18–33", label: "Code it" },
  { range: "33–40", label: "Trace + edge cases" },
  { range: "40–45", label: "Complexity + follow-ups" },
];

// --- live (phase rail + server timer) / scoring ---

function Live({ session, onNewMock }: { session: MockSession; onNewMock: () => void }) {
  const qc = useQueryClient();
  const [scoring, setScoring] = useState(false);

  const scoreM = useMutation({
    mutationFn: ({ scores, notes }: { scores: RubricScores; notes: string }) => scoreMock(session.id, scores, notes),
    onSuccess: (scored) => {
      qc.setQueryData(["mock", session.id], scored);
      qc.invalidateQueries({ queryKey: ["mock", "trend"] });
    },
  });

  if (scoring) {
    return (
      <Scoring
        session={session}
        submitting={scoreM.isPending}
        failed={scoreM.isError}
        onSubmit={(scores, notes) => scoreM.mutate({ scores, notes })}
        onBack={() => setScoring(false)}
      />
    );
  }
  return <LiveRail session={session} onFinish={() => setScoring(true)} onNewMock={onNewMock} />;
}

function LiveRail({ session, onFinish, onNewMock }: { session: MockSession; onFinish: () => void; onNewMock: () => void }) {
  const elapsed = useElapsed(session.startedAt);
  const [draft, setDraft] = useState("");
  const curIdx = currentPhaseIndex(session.rail.phases, elapsed);
  const current: MockPhase =
    session.rail.phases[curIdx] ??
    session.rail.phases[0] ?? { index: 0, label: "", range: "", startMin: 0, endMin: 45, grow: 45, prompt: "", state: "current" };
  const title = session.problem?.title ?? `Problem ${session.problemId || session.setId}`;
  const pattern = session.problem?.pattern;
  const elapsedMin = elapsed / 60;

  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 16, marginBottom: 18, flexWrap: "wrap" }}>
        <div>
          <div className="xl-eyebrow" style={{ color: "var(--ds-err)" }}>● Live mock · {session.setId}</div>
          <h1 style={{ marginTop: 4, fontSize: 20, fontWeight: 700 }}>
            {title}
            {pattern ? ` · ${pattern}` : ""}
          </h1>
        </div>
        <div style={{ marginLeft: "auto", display: "flex", alignItems: "center", gap: 14, flexWrap: "wrap" }}>
          <span className="ds-badge ds-badge--violet">
            <Icon name="mic" className="xl-ico--sm" /> Talk aloud · no notes
          </span>
          <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "8px 14px", background: "var(--ds-panel)", border: "1px solid var(--ds-line-2)", borderRadius: 10 }}>
            <Icon name="clock" className="xl-ico--sm" />
            <span className="xl-timer__t" style={{ fontSize: 22, color: clockColor(elapsedMin) }}>{formatMMSS(elapsed)}</span>
            <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-muted)" }}>/ 45:00</span>
          </div>
        </div>
      </div>

      {/* phase rail */}
      <div style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", gap: 4 }}>
          {session.rail.phases.map((p) => {
            const state = p.index < curIdx ? "past" : p.index === curIdx ? "current" : "upcoming";
            return (
              <div key={p.index} style={{ flex: p.grow, height: 44, borderRadius: 8, padding: "7px 10px", overflow: "hidden", ...phaseSegStyle(state) }}>
                <div className="ds-mono" style={{ fontSize: 10, opacity: 0.8 }}>{p.range}</div>
                <div style={{ fontSize: 11.5, fontWeight: 600, whiteSpace: "nowrap", overflow: "hidden", textOverflow: "ellipsis" }}>{p.label}</div>
              </div>
            );
          })}
        </div>
      </div>

      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 340px", gap: 18, alignItems: "start" }}>
        <div className="xl-panel" style={{ overflow: "hidden" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "9px 14px", borderBottom: "1px solid var(--ds-line)" }}>
            <Icon name="code" className="xl-ico--sm" />
            <span className="ds-mono" style={{ fontSize: 12 }}>solution.go</span>
            <span style={{ marginLeft: "auto", fontSize: 11, color: "var(--ds-muted)" }}>no autocomplete · interview mode</span>
          </div>
          <textarea
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            spellCheck={false}
            aria-label="Interview editor"
            placeholder={"// interview mode — narrate as you type\nfunc solve() {\n\n}"}
            className="ds-mono"
            style={{ width: "100%", minHeight: 300, resize: "vertical", background: "var(--ds-inset)", color: "var(--ds-text)", border: "none", padding: 14, fontSize: 12.5, lineHeight: 1.6, display: "block" }}
          />
        </div>

        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div className="ds-card ds-card--teal" style={{ padding: 16 }}>
            <div className="ds-mono" style={{ fontSize: 11, color: "var(--ds-teal)", textTransform: "uppercase", letterSpacing: ".5px", marginBottom: 8 }}>
              Now · {current.label}
            </div>
            <p style={{ fontSize: 13, color: "var(--ds-text)", lineHeight: 1.55, margin: 0 }}>{current.prompt}</p>
          </div>
          <div className="ds-card" style={{ padding: 14, fontSize: 12, color: "var(--ds-dim)", lineHeight: 1.6 }}>
            Phase {curIdx + 1} of 6 · {formatMMSS(session.rail.remainingSeconds > 0 ? clampRemaining(elapsed) : 0)} left on the clock.
            {session.rail.overtime || elapsed >= MOCK_TOTAL_SECS ? " Time’s up — wrap and score." : ""}
          </div>
          <button type="button" className="ds-btn ds-btn--primary ds-btn--block" onClick={onFinish}>
            <Icon name="flag" className="xl-ico--sm" /> Finish &amp; score
          </button>
          <button type="button" className="ds-btn ds-btn--ghost ds-btn--sm" onClick={onNewMock}>
            Abandon &amp; restart
          </button>
        </div>
      </div>
    </div>
  );
}

function Scoring({
  session,
  submitting,
  failed,
  onSubmit,
  onBack,
}: {
  session: MockSession;
  submitting: boolean;
  failed: boolean;
  onSubmit: (scores: RubricScores, notes: string) => void;
  onBack: () => void;
}) {
  const [scores, setScores] = useState<RubricScores>(() =>
    Object.fromEntries(RUBRIC_DIMENSIONS.map((d) => [d.key, 3])),
  );
  const [notes, setNotes] = useState("");
  const total = RUBRIC_DIMENSIONS.reduce((t, d) => t + (scores[d.key] ?? 3), 0);
  const set = (key: string, v: number) => setScores((prev) => ({ ...prev, [key]: v }));

  return (
    <div>
      <PageHeader eyebrow={`Score your mock · ${session.setId}`} title="Rubric" sub="Rate each dimension 1–5, honestly. The server totals your /35 and charts the trend." />
      <div className="xl-panel">
        <div className="xl-panel__b" style={{ padding: 18 }}>
          <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
            {RUBRIC_DIMENSIONS.map((d) => {
              const v = scores[d.key] ?? 3;
              return (
                <div key={d.key} style={{ display: "flex", alignItems: "center", gap: 14, flexWrap: "wrap" }}>
                  <span style={{ width: 170, fontSize: 12.5, color: "var(--ds-dim)", flex: "none" }}>{d.name}</span>
                  <div className="ds-seg" role="group" aria-label={d.name}>
                    {[1, 2, 3, 4, 5].map((n) => (
                      <button
                        key={n}
                        type="button"
                        className={`ds-seg__btn${v === n ? " ds-seg__btn--on" : ""}`}
                        aria-pressed={v === n}
                        onClick={() => set(d.key, n)}
                      >
                        {n}
                      </button>
                    ))}
                  </div>
                  <span className="ds-mono" style={{ fontSize: 13, fontWeight: 600, width: 22, textAlign: "right", color: dimColor(v) }}>{v}</span>
                </div>
              );
            })}
          </div>

          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Notes — what to fix next mock (optional)"
            aria-label="Mock notes"
            style={{ width: "100%", minHeight: 70, resize: "vertical", background: "var(--ds-input-bg)", color: "var(--ds-text)", border: "1px solid var(--ds-line-2)", borderRadius: 8, padding: 12, fontSize: 12.5, lineHeight: 1.6, marginTop: 18 }}
          />

          {failed && <div className="ds-field__error" style={{ marginTop: 10 }}>Couldn’t save the score. Try again.</div>}

          <div style={{ display: "flex", alignItems: "center", gap: 12, marginTop: 16, flexWrap: "wrap" }}>
            <div style={{ display: "flex", alignItems: "baseline", gap: 6 }}>
              <span className="ds-mono" style={{ fontSize: 26, fontWeight: 700, color: "var(--ds-teal)" }}>{total}</span>
              <span style={{ color: "var(--ds-muted)", fontSize: 13 }}>/ 35</span>
            </div>
            <div style={{ marginLeft: "auto", display: "flex", gap: 8 }}>
              <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={onBack} disabled={submitting}>
                Back
              </button>
              <button type="button" className="ds-btn ds-btn--primary" onClick={() => onSubmit(scores, notes)} disabled={submitting}>
                {submitting ? "Scoring…" : "Submit rubric — score /35"}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// --- results (radar + trend + per-dimension meters) ---

function Results({ session, onNewMock }: { session: MockSession; onNewMock: () => void }) {
  const trendQ = useMockTrend();
  const total = session.total35 ?? 0;
  const targets = session.targets;
  const scores = session.dimensions.map((d) => d.score);

  const nearestTarget = useMemo(() => {
    if (total >= targets.pre) return { label: "pre-interview target of " + targets.pre, met: true };
    if (total >= targets.w15) return { label: "Week-15 target of " + targets.w15, met: true };
    if (total >= targets.w13) return { label: "Week-13 target of " + targets.w13, met: true };
    return { label: "Week-13 target of " + targets.w13, met: false };
  }, [total, targets]);

  return (
    <div>
      <PageHeader
        eyebrow={`Mock complete · ${session.setId}${session.problem ? " · " + session.problem.title : ""}`}
        title="Rubric & score"
        sub={session.date}
        actions={
          <div style={{ display: "flex", gap: 8 }}>
            <button type="button" className="ds-btn ds-btn--secondary" onClick={onNewMock}>
              <Icon name="refresh" className="xl-ico--sm" /> New mock
            </button>
            <Link className="ds-btn ds-btn--secondary" to="/dsa/mistakes">
              Log takeaways
            </Link>
          </div>
        }
      />

      <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 20, alignItems: "start", marginBottom: 20 }}>
        <div className="xl-panel">
          <div className="xl-panel__b" style={{ display: "flex", gap: 18, alignItems: "center", flexWrap: "wrap" }}>
            <RubricRadar scores={scores} />
            <div style={{ flex: 1, minWidth: 160 }}>
              <div style={{ display: "flex", alignItems: "baseline", gap: 6 }}>
                <span className="ds-mono" style={{ fontSize: 44, fontWeight: 700, color: "var(--ds-teal)" }}>{total}</span>
                <span style={{ color: "var(--ds-muted)", fontSize: 16 }}>/ 35</span>
              </div>
              <div style={{ fontSize: 12.5, color: "var(--ds-dim)", marginTop: 6, lineHeight: 1.5 }}>
                {nearestTarget.met ? "At or above the " : "Below the "}
                <b style={{ color: "var(--ds-text)" }}>{nearestTarget.label}</b>. Your lowest axes are the ones to lift next.
              </div>
            </div>
          </div>
        </div>

        <div className="xl-panel">
          <div className="xl-panel__b">
            <div style={{ fontSize: 12, color: "var(--ds-dim)", marginBottom: 12 }}>Score trend vs targets</div>
            {trendQ.isLoading && <div className="xl-skel" style={{ height: 176 }} />}
            {trendQ.isError && <div className="xl-mut" style={{ fontSize: 12 }}>Couldn’t load the trend.</div>}
            {trendQ.data && <TrendChart trend={trendQ.data} />}
          </div>
        </div>
      </div>

      <div className="xl-panel">
        <div className="xl-panel__b">
          <div style={{ fontSize: 12, color: "var(--ds-dim)", marginBottom: 14 }}>Seven dimensions · 1–5 each</div>
          <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "14px 28px" }}>
            {session.dimensions.map((d) => (
              <div key={d.key} style={{ display: "flex", alignItems: "center", gap: 12 }}>
                <span style={{ width: 150, fontSize: 12.5, color: "var(--ds-dim)", flex: "none" }}>{d.name}</span>
                <div className="ds-meter" style={{ flex: 1 }}>
                  <div className="ds-meter__fill" style={{ width: `${(d.score / 5) * 100}%`, background: dimColor(d.score) }} />
                </div>
                <span className="ds-mono" style={{ fontSize: 13, fontWeight: 600, width: 34, textAlign: "right", color: dimColor(d.score) }}>{d.score}</span>
              </div>
            ))}
          </div>
          {session.notes && (
            <div style={{ marginTop: 16, paddingTop: 14, borderTop: "1px solid var(--ds-line)", fontSize: 12.5, color: "var(--ds-dim)", lineHeight: 1.6 }}>
              <b style={{ color: "var(--ds-text)" }}>Notes ·</b> {session.notes}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// --- radar ---

const RADAR = { cx: 120, cy: 116, r: 92, n: 7 };

function radarPoint(i: number, rad: number): [number, number] {
  const a = -Math.PI / 2 + (i * 2 * Math.PI) / RADAR.n;
  return [RADAR.cx + rad * Math.cos(a), RADAR.cy + rad * Math.sin(a)];
}

function ringPoints(frac: number): string {
  return RUBRIC_DIMENSIONS.map((_, i) => radarPoint(i, RADAR.r * frac).map((v) => v.toFixed(1)).join(",")).join(" ");
}

function RubricRadar({ scores }: { scores: number[] }) {
  const axes = RUBRIC_DIMENSIONS.map((_, i) => {
    const [x, y] = radarPoint(i, RADAR.r);
    return `M${RADAR.cx},${RADAR.cy} L${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" ");
  const data = scores.map((s, i) => radarPoint(i, (RADAR.r * s) / 5).map((v) => v.toFixed(1)).join(",")).join(" ");
  return (
    <svg width="230" height="224" viewBox="0 0 240 232" style={{ flex: "none" }} role="img" aria-label="Rubric radar">
      {[0.2, 0.4, 0.6, 0.8, 1].map((f) => (
        <polygon key={f} points={ringPoints(f)} fill="none" stroke={f === 1 ? "#232a35" : "#1b2028"} strokeWidth="1" />
      ))}
      <path d={axes} stroke="#232a35" strokeWidth="1" />
      <polygon points={data} fill="rgba(53,208,192,.18)" stroke="var(--ds-teal)" strokeWidth="2" strokeLinejoin="round" />
    </svg>
  );
}

// --- trend chart ---

// The y-domain spans the full achievable /35 range (7 = all-1s … 35 = all-5s) so every
// possible total plots at a distinct height — a 35 must read higher than a 32, and the
// sub-14 floor must not pin. The W13/W15/pre target lines (24/28/30) sit inside it.
const TREND = { w: 380, h: 176, top: 16, bottom: 150, scaleMin: 7, scaleMax: 35, padX: 30 };

function scoreToY(score: number): number {
  const clamped = Math.max(TREND.scaleMin, Math.min(TREND.scaleMax, score));
  const frac = (clamped - TREND.scaleMin) / (TREND.scaleMax - TREND.scaleMin);
  return TREND.bottom - frac * (TREND.bottom - TREND.top);
}

function pointX(i: number, n: number): number {
  if (n <= 1) return TREND.w / 2;
  return TREND.padX + (i * (TREND.w - 2 * TREND.padX)) / (n - 1);
}

function TrendChart({ trend }: { trend: MockTrend }) {
  const pts = trend.points;
  const n = pts.length;
  const coords = pts.map((p, i) => ({ x: pointX(i, n), y: scoreToY(p.total35), p }));
  const poly = coords.map((c) => `${c.x.toFixed(1)},${c.y.toFixed(1)}`).join(" ");
  const targetLines = [
    { y: scoreToY(trend.targets.pre), color: "rgba(87,211,154,.5)", label: `${trend.targets.pre} pre-interview`, legend: "var(--ds-ok)" },
    { y: scoreToY(trend.targets.w15), color: "rgba(240,180,41,.5)", label: `${trend.targets.w15} W15`, legend: "var(--ds-warn)" },
    { y: scoreToY(trend.targets.w13), color: "rgba(155,140,240,.55)", label: `${trend.targets.w13} W13`, legend: "var(--ds-violet)" },
  ];

  return (
    <div>
      <svg width="100%" height={TREND.h} viewBox={`0 0 ${TREND.w} ${TREND.h}`} preserveAspectRatio="none" style={{ overflow: "visible" }} role="img" aria-label="Score trend">
        {targetLines.map((t) => (
          <line key={t.label} x1="0" y1={t.y} x2={TREND.w} y2={t.y} stroke={t.color} strokeWidth="1" strokeDasharray="4 4" />
        ))}
        {n > 1 && <polyline points={poly} fill="none" stroke="var(--ds-teal)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />}
        {coords.map((c, i) => (
          <circle key={c.p.mockId} cx={c.x} cy={c.y} r={i === n - 1 ? 5 : 4} fill="var(--ds-teal)" stroke={i === n - 1 ? "var(--ds-bg)" : undefined} strokeWidth={i === n - 1 ? 2 : undefined} />
        ))}
      </svg>
      <div style={{ display: "flex", justifyContent: "space-between", fontSize: 10.5, color: "var(--ds-muted)", marginTop: 2 }} className="ds-mono">
        <span>{n > 0 ? formatShortDate(pts[0]!.date) : ""}</span>
        <span>{n > 1 ? formatShortDate(pts[n - 1]!.date) : ""}</span>
      </div>
      <div style={{ display: "flex", gap: 14, marginTop: 12, fontSize: 11, flexWrap: "wrap" }}>
        {targetLines.map((t) => (
          <span key={t.label} style={{ color: t.legend }}>— {t.label}</span>
        ))}
      </div>
    </div>
  );
}

// --- shared ---

function PageHeader({ eyebrow, title, sub, actions }: { eyebrow: string; title: string; sub?: string; actions?: ReactNode }) {
  return (
    <div className="xl-page-h" style={{ display: "flex", alignItems: "flex-start", gap: 16, borderBottom: "1px solid var(--ds-line)", paddingBottom: 16, marginBottom: 20 }}>
      <div style={{ flex: 1 }}>
        <div className="xl-eyebrow">{eyebrow}</div>
        <h1 style={{ marginTop: 6, fontSize: 26, fontWeight: 700, letterSpacing: "-.3px" }}>{title}</h1>
        {sub && <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)", maxWidth: 640 }}>{sub}</p>}
      </div>
      {actions}
    </div>
  );
}

function MockSkeleton() {
  return (
    <div>
      <div className="xl-skel" style={{ height: 56, marginBottom: 18 }} />
      <div className="xl-skel" style={{ height: 44, marginBottom: 16 }} />
      <div style={{ display: "grid", gridTemplateColumns: "minmax(0,1fr) 340px", gap: 18, alignItems: "start" }}>
        <div className="xl-skel" style={{ height: 340 }} />
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div className="xl-skel" style={{ height: 120 }} />
          <div className="xl-skel" style={{ height: 80 }} />
        </div>
      </div>
    </div>
  );
}

// --- pure helpers ---

// useElapsed ticks the elapsed seconds since the server-set startedAt, clamped to the
// 45-minute window. The anchor is server time (R-MK1: server-authoritative), so a
// refresh/return resumes the same count and past-deadline clamps at 45:00.
function useElapsed(startedAtISO: string): number {
  const start = new Date(startedAtISO).getTime();
  const [elapsed, setElapsed] = useState(0);
  useEffect(() => {
    const tick = () => setElapsed(Math.max(0, Math.min(MOCK_TOTAL_SECS, Math.round((Date.now() - start) / 1000))));
    tick();
    const iv = window.setInterval(tick, 1000);
    return () => window.clearInterval(iv);
  }, [start]);
  return elapsed;
}

// currentPhaseIndex mirrors the server rail: the first phase whose end boundary the
// elapsed time has not reached, or the last phase once the window is exhausted.
function currentPhaseIndex(phases: MockPhase[], elapsedSecs: number): number {
  for (const p of phases) {
    if (elapsedSecs < p.endMin * 60) return p.index;
  }
  return phases.length > 0 ? phases.length - 1 : 0;
}

function clampRemaining(elapsedSecs: number): number {
  return Math.max(0, MOCK_TOTAL_SECS - elapsedSecs);
}

function phaseSegStyle(state: "past" | "current" | "upcoming"): CSSProperties {
  if (state === "past") return { background: "rgba(87,211,154,.12)", color: "var(--ds-ok)", border: "1px solid rgba(87,211,154,.3)" };
  if (state === "current") return { background: "rgba(53,208,192,.16)", color: "var(--ds-teal)", border: "1px solid var(--ds-teal)" };
  return { background: "var(--ds-panel)", color: "var(--ds-muted)", border: "1px solid var(--ds-line)" };
}

function dimColor(score: number): string {
  return score >= 4 ? "var(--ds-ok)" : score === 3 ? "var(--ds-warn)" : "var(--ds-err)";
}

function clockColor(elapsedMin: number): string {
  return elapsedMin >= 43 ? "var(--ds-err)" : elapsedMin >= 40 ? "var(--ds-warn)" : "var(--ds-teal)";
}

function formatMMSS(secs: number): string {
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  return `${String(m).padStart(2, "0")}:${String(s).padStart(2, "0")}`;
}

function formatShortDate(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric" });
}
