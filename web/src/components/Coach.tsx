import { useEffect, useId, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import type { IconName } from "./Icon";
import { Icon } from "./Icon";
import { useCourse } from "../lib/course";
import {
  CoachChatError,
  catalogModelsFor,
  coachFeatureDefault,
  providerLabel,
  splitTruncationNote,
  streamCoachChat,
  useCoachKey,
  useCoachModels,
  useCoachThread,
  usePutCoachKey,
  type CoachChatBody,
  type CoachErrorReason,
  type CoachMessage,
  type CoachMode,
} from "../lib/settings";
import { CoachModelMenu } from "./CoachModelSwitcher";

/** The board's name for the BYO coach (AB01 naming reference, ADR-0031 §7). Never
 *  shortened to "AI" alone. */
const COACH_NAME = "Your AI coach (your key)";

/** The fixed user turn AB01 F9's Continue sends through the normal send path. */
export const CONTINUE_TURN = "Continue from where you stopped.";

/** The codes the gateway answers BEFORE it records a D27 assist (AB01 F2): the confirm
 *  itself, a failed record or state lookup, the mock lock and the L18 admission probe. Any
 *  other outcome of an acknowledged send means the assist was recorded — a provider failure
 *  after it still leaves the attempt capped (t5 §9: the learner chose to ask). */
const BEFORE_ASSIST_RECORD = new Set([
  "assist_confirm_required",
  "assist_unavailable",
  "coach_state_unavailable",
  "coach_paused",
  "coach_busy",
  "coach_rate_limited",
  "coach_daily_cap",
]);

/** A turn the panel may have to send again: the message, whether it came from the composer
 *  (so it is the learner's draft), and the D27 acknowledgement it carried. */
interface PendingTurn {
  message: string;
  fromComposer: boolean;
  ack?: string;
}

/** The F2 card's state: the turn waiting on the learner's answer and the attempt it caps. */
interface AssistConfirm extends PendingTurn {
  attemptId: string;
  problemId?: string;
}

/** A notice for a turn the server refused before sending anything (AB01 F4 and F8). */
type CoachNotice =
  | { kind: "assist_unavailable" | "coach_state_unavailable"; retry: PendingTurn }
  | { kind: "rate" | "busy"; until: number }
  | { kind: "daily"; until: number; at: string };

/** A transcript row: a persisted turn, plus the live `done.truncated` flag for a reply
 *  streamed in this session (AB01 F9). */
type PanelMessage = CoachMessage & { truncated?: boolean };

/**
 * Coach is the persistent AI-coach panel (Problem.dc.html / Concept.dc.html, AB01): a FAB
 * that opens a side panel with the current-page context chip, the mode the server decided,
 * the thread history, and a live chat that streams the reply token-by-token (SSE). It runs
 * on the user's own provider key — with no key it shows an empty state that routes to
 * Settings.
 *
 * The gateway owns the mode gate (m1-07, ADR-0031 §7): it reports the mode on the thread
 * (`gate`) and on every chat response (`X-Coach-Mode`), asks for the D27 confirm, fails
 * closed, pauses the coach during a live mock, and enforces L18. This panel only reports
 * what the server returned; it never gates the coach itself.
 */
export function Coach() {
  const [open, setOpen] = useState(false);
  const [refocusFab, setRefocusFab] = useState(false);
  const fabRef = useRef<HTMLButtonElement>(null);
  const closeRef = useRef<HTMLButtonElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const qc = useQueryClient();
  const ctx = useCoachContext(pathname);

  const keyQuery = useCoachKey(open);
  // The coach answers with the key the `coach` FEATURE default points at, so usability
  // tracks that one (AB01 F15: "no key row, or the default key for coach is missing").
  const coachDefault = coachFeatureDefault(keyQuery.data, "coach");
  const key = keyQuery.data?.keys?.find((k) => k.provider === coachDefault?.provider);
  const usable = !!key && key.enabled;
  const thread = useCoachThread(ctx.context, open && usable);
  const gate = thread.data?.gate;

  const [messages, setMessages] = useState<PanelMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  // The last provider refusal AB01 gives a line of its own (quota, rate limit, region, a
  // model this key may not use), with the turn it refused (F10's Retry sends it again). It
  // clears on the next send: it describes one attempt.
  const [failure, setFailure] = useState<{ reason: CoachErrorReason; message: string } | null>(null);
  // The mode from the latest chat response's X-Coach-Mode; it overrides the thread's
  // `gate.mode` until the thread is next (re)loaded, which is the server's fresher answer.
  const [modeOverride, setModeOverride] = useState<CoachMode | null>(null);
  // An acknowledged send got past the assist record (AB01 F3), before the thread says so.
  const [assistRecorded, setAssistRecorded] = useState(false);
  const [confirm, setConfirm] = useState<AssistConfirm | null>(null);
  const [notice, setNotice] = useState<CoachNotice | null>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  // Load the server thread when it (re)loads or the page context changes, unless a live
  // exchange is in flight (don't clobber the streaming reply). A fresh thread also carries
  // the server's current mode, so any per-response override gives way to it.
  useEffect(() => {
    if (!sending && thread.data) {
      setMessages(thread.data.messages);
      // A thread without `gate` means the state lookup failed — unknown, not "no mode" —
      // so the last chat's X-Coach-Mode stands until a thread carries one again.
      if (thread.data.gate) setModeOverride(null);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [thread.data, ctx.context]);

  // A new page context is a new thread with its own gate: drop every per-thread state.
  useEffect(() => {
    setModeOverride(null);
    setAssistRecorded(false);
    setConfirm(null);
    setNotice(null);
    setFailure(null);
  }, [ctx.context]);

  // Keep the transcript scrolled to the newest message (scrollTo is absent in jsdom).
  useEffect(() => {
    bodyRef.current?.scrollTo?.({ top: bodyRef.current.scrollHeight });
  }, [messages, open, notice, confirm]);

  // Closing the panel returns focus to the FAB (AB01 F1 A11y).
  useEffect(() => {
    if (!open && refocusFab) {
      fabRef.current?.focus();
      setRefocusFab(false);
    }
  }, [open, refocusFab]);

  const mode: CoachMode | null = modeOverride ?? gate?.mode ?? null;
  const paused = mode === "locked";
  const recorded = assistRecorded || !!gate?.attempt?.coachAssistAt;
  const dailyCapped = notice?.kind === "daily";
  // L18 and the confirm hold the composer: Send is aria-disabled until they clear.
  const held = !!confirm || paused || notice?.kind === "rate" || notice?.kind === "busy" || dailyCapped;
  const inputDisabled = sending || !!confirm || paused || dailyCapped;

  // While paused, focus lands on Close (AB01 F5 A11y).
  useEffect(() => {
    if (open && usable && paused) closeRef.current?.focus();
  }, [open, usable, paused]);

  function closePanel() {
    setOpen(false);
    setRefocusFab(true);
  }

  /** markAssistRecorded shows F3's note and refreshes the Problem workspace, whose HUD chip
   *  reads the attempt's `coachAssistAt` from GET /problems/{id} (prefix match: the
   *  workspace caches it as ["problem", id, mode]). */
  function markAssistRecorded(problemId?: string) {
    setAssistRecorded(true);
    if (problemId) void qc.invalidateQueries({ queryKey: ["problem", problemId] });
  }

  /** unsend undoes a turn the server refused before sending anything: the optimistic
   *  bubbles go, and the message is kept as the composer's draft. */
  function unsend(turn: PendingTurn) {
    setMessages((prev) => {
      let next = prev;
      const last = next[next.length - 1];
      if (last?.role === "assistant" && last.content === "") next = next.slice(0, -1);
      const mine = next[next.length - 1];
      if (mine?.role === "user" && mine.content === turn.message) next = next.slice(0, -1);
      return next;
    });
    setInput((draft) => (turn.fromComposer || draft.trim() === "" ? turn.message : draft));
  }

  async function send(text: string, opts: { fromComposer: boolean; ack?: string }) {
    const message = text.trim();
    if (!message || sending) return;
    if (!usable) {
      navigate("/settings?tab=coach");
      return;
    }
    const turn: PendingTurn = { message, fromComposer: opts.fromComposer, ack: opts.ack };
    if (opts.fromComposer) setInput("");
    setSending(true);
    setFailure(null);
    setNotice(null);
    setMessages((prev) => [...prev, { role: "user", content: message }, { role: "assistant", content: "" }]);

    const appendDelta = (delta: string) =>
      setMessages((prev) => {
        const next = prev.slice();
        const last = next[next.length - 1];
        if (last && last.role === "assistant") next[next.length - 1] = { ...last, content: last.content + delta };
        return next;
      });

    const chatBody: CoachChatBody = {
      context: ctx.context,
      kind: ctx.kind,
      label: ctx.label,
      problemId: ctx.problemId,
      problemTitle: ctx.problemTitle,
      pattern: ctx.pattern,
      stage: ctx.stage,
      recentOutcome: ctx.recentOutcome,
      message,
      ...(opts.ack ? { assist_ack: opts.ack } : {}),
    };

    try {
      const result = await streamCoachChat(chatBody, appendDelta, { onMode: setModeOverride });
      if (result.truncated) {
        setMessages((prev) => {
          const next = prev.slice();
          const last = next[next.length - 1];
          if (last && last.role === "assistant") next[next.length - 1] = { ...last, truncated: true };
          return next;
        });
      }
      if (opts.ack) markAssistRecorded(ctx.problemId);
      // Refresh the persisted thread so a later reopen reflects the server's copy.
      void qc.invalidateQueries({ queryKey: ["coach-thread", ctx.context] });
    } catch (err) {
      if (err instanceof CoachChatError && opts.ack && !BEFORE_ASSIST_RECORD.has(err.code)) {
        markAssistRecorded(err.problemId ?? ctx.problemId);
      }
      handleFailure(err, turn);
    } finally {
      setSending(false);
    }
  }

  function handleFailure(err: unknown, turn: PendingTurn) {
    if (err instanceof CoachChatError) {
      const now = Date.now();
      switch (err.code) {
        case "assist_confirm_required": {
          // F2: nothing was sent. Ask before capping this attempt at Assisted.
          const attemptId = err.attemptId ?? gate?.attempt?.attemptId;
          if (!attemptId) break; // no attempt to acknowledge: fall through to the generic note
          unsend(turn);
          setConfirm({ ...turn, attemptId, problemId: err.problemId ?? ctx.problemId });
          return;
        }
        case "assist_unavailable":
        case "coach_state_unavailable":
          // F4: the gateway failed closed; nothing was forwarded, persisted or counted.
          unsend(turn);
          setNotice({ kind: err.code, retry: turn });
          requestAnimationFrame(() => inputRef.current?.focus());
          return;
        case "coach_paused":
          // F5: a live mock (or touch) locks the coach; the gateway also says so in
          // X-Coach-Mode, but a 409 coach_paused is enough on its own.
          unsend(turn);
          setModeOverride("locked");
          return;
        case "coach_rate_limited":
          unsend(turn);
          setNotice({ kind: "rate", until: now + (err.retryAfter ?? 60) * 1000 });
          return;
        case "coach_busy":
          unsend(turn);
          setNotice({ kind: "busy", until: now + (err.retryAfter ?? 5) * 1000 });
          return;
        case "coach_daily_cap": {
          // Retry-After is the seconds to the next UTC midnight; show that instant on the
          // learner's own clock (AB01 F8c: the day is the UTC day).
          const until = now + (err.retryAfter ?? secondsToUtcMidnight(now)) * 1000;
          unsend(turn);
          setNotice({ kind: "daily", until, at: localClock(until) });
          return;
        }
      }
    }

    if (err instanceof CoachChatError && err.routesToSettings) {
      // Only an auth failure disables a key, which is what routes back to Settings (F15b).
      void qc.invalidateQueries({ queryKey: ["coach-key"] });
      navigate("/settings?tab=coach");
    } else if (err instanceof CoachChatError && hasFailureNote(err.reason)) {
      // A refusal AB01 has a line for: show that line instead of appending to the reply,
      // and drop the empty assistant bubble the send optimistically opened. The key stays
      // enabled — quota, a rate limit, a region and a model this key may not use are all
      // things the learner fixes without re-entering a working key (the v1 bug this
      // taxonomy exists to end).
      setFailure({ reason: err.reason, message: turn.message });
      setMessages((prev) => (prev[prev.length - 1]?.role === "assistant" && prev[prev.length - 1]!.content === "" ? prev.slice(0, -1) : prev));
    } else {
      // Anything else is a transient failure, noted on the reply itself (v1 behaviour).
      const note = "⚠️ The coach couldn’t reply just now. Please try again.";
      setMessages((prev) => {
        const next = prev.slice();
        const last = next[next.length - 1];
        if (last && last.role === "assistant") {
          next[next.length - 1] = { ...last, content: last.content === "" ? note : `${last.content}\n\n${note}` };
        }
        return next;
      });
    }
  }

  /** submit sends the composer's draft, unless a hold (F2, F5, F8) is up. Enter in the
   *  composer never confirms F2: the composer is disabled while the card shows. */
  function submit() {
    if (held) return;
    void send(input, { fromComposer: true });
  }

  /** sendTurn sends a turn that didn't come from the composer (a suggestion, F9's
   *  Continue) through the same path, so it meets the same gate and limits. */
  function sendTurn(text: string) {
    if (held) return;
    void send(text, { fromComposer: false });
  }

  /** retryProvider re-sends the turn the provider refused (F10's Retry): it is still the
   *  last bubble in the thread, so it moves rather than repeats. */
  function retryProvider() {
    if (!failure || held) return;
    const message = failure.message;
    setMessages((prev) => {
      const last = prev[prev.length - 1];
      return last?.role === "user" && last.content === message ? prev.slice(0, -1) : prev;
    });
    void send(message, { fromComposer: false });
  }

  /** retryClosed re-sends the kept draft after F4 (with the D27 ack in case a). */
  function retryClosed(retry: PendingTurn) {
    const draft = input.trim();
    void send(draft || retry.message, { fromComposer: draft !== "", ack: retry.ack });
  }

  function askConfirmed() {
    if (!confirm) return;
    const turn = confirm;
    setConfirm(null);
    void send(turn.message, { fromComposer: turn.fromComposer, ack: turn.attemptId });
  }

  function cancelConfirm() {
    setConfirm(null);
    requestAnimationFrame(() => inputRef.current?.focus());
  }

  if (!open) {
    return (
      <button ref={fabRef} className="xl-fab" onClick={() => setOpen(true)} aria-label={`Open ${COACH_NAME}`}>
        <Icon name="spark" className="xl-ico--lg" />
      </button>
    );
  }

  const lastIndex = messages.length - 1;
  const showIntro = messages.length === 0 && !paused;

  return (
    <aside
      className="xl-coach"
      aria-label={COACH_NAME}
      onKeyDown={(e) => {
        // Esc closes the panel (AB01 F1) — unless something inside handled it first (F2's
        // card cancels; an open model list closes itself).
        if (e.key === "Escape" && !e.defaultPrevented) {
          e.preventDefault();
          closePanel();
        }
      }}
    >
      <div className="xl-coach__top">
        <div className="xl-coach__title">
          <Icon name="spark" /> Your AI coach <span className="xl-coach__sub">(your key)</span>
        </div>
        <span className="xl-coach__flex" />
        <button ref={closeRef} className="ds-iconbtn" onClick={closePanel} aria-label="Close coach">
          <Icon name="close" />
        </button>
      </div>

      <div className="xl-coach__ctx">
        <Icon name={ctx.chipIcon} className="xl-ico--sm" /> Context · {ctx.label}
      </div>

      {usable && (mode || recorded) && (
        <div className="xl-coach__mode">
          {mode && <ModeChip mode={mode} />}
          {recorded && (
            <span className="ds-badge ds-badge--warn" role="status">
              Coach used on this attempt · capped at Assisted
            </span>
          )}
        </div>
      )}

      <div className="xl-coach__body xl-scroll" ref={bodyRef} role="log" aria-live="polite" aria-relevant="additions text">
        {keyQuery.isLoading && <div className="xl-coach__loading">Loading your coach…</div>}

        {!keyQuery.isLoading && !usable && <CoachEmptyState disabled={!!key && !key.enabled} onOpenSettings={() => navigate("/settings?tab=coach")} />}

        {usable && (
          <>
            {mode === "general" && (
              // F7: static copy on every general-mode panel. The guard is server-side (the
              // gateway names the live items off-limits in the prompt); this only says so.
              <div className="xl-coach__note xl-coach__note--info">
                <Icon name="lock" className="xl-ico--sm" />
                <div className="xl-coach__note-txt">Problems you're mid-attempt on stay off-limits in general chat. Ask about them on their own page.</div>
              </div>
            )}

            {paused && (
              <div className="xl-coach__note" role="status">
                <Icon name="pause" className="xl-ico--sm" />
                <div className="xl-coach__note-txt">The coach is paused while a revision touch or mock is live. It's back when you finish.</div>
              </div>
            )}

            {messages.map((m, i) => {
              if (m.role === "user") {
                return (
                  <div key={i} className="xl-coach__msg xl-coach__msg--me xl-coach__msg--pre">
                    {m.content}
                  </div>
                );
              }
              const { text, cut } = splitTruncationNote(m.content);
              return (
                <div key={i} className="xl-coach__msg xl-coach__msg--bot xl-coach__msg--pre">
                  {text || (sending && i === lastIndex ? "…" : "")}
                  {(cut || m.truncated) && (
                    // F9: the marker is text inside the message, read in order; Continue is a
                    // real button next in tab order, on the latest reply only.
                    <div className="xl-coach__cut">
                      <Icon name="minus" className="xl-ico--sm" /> Reply cut short (length limit)
                      <span className="xl-coach__flex" />
                      {i === lastIndex && (
                        <button
                          type="button"
                          className="ds-btn ds-btn--secondary ds-btn--sm"
                          aria-disabled={held || undefined}
                          disabled={sending}
                          onClick={() => sendTurn(CONTINUE_TURN)}
                        >
                          Continue
                        </button>
                      )}
                    </div>
                  )}
                </div>
              );
            })}

            {showIntro && (
              <>
                <div className="xl-coach__msg xl-coach__msg--bot">
                  {mode === "attempt" ? (
                    // F1: the attempt-mode intro.
                    <>
                      You're mid-attempt on {ctx.short}, so I'll stick to questions and hints — no pattern names or solution steps until you've
                      concluded.
                    </>
                  ) : (
                    <>
                      I read the current page and coach you — Socratically while you’re attempting, and as a reviewer once you’ve solved. Ask
                      me anything about {ctx.short}.
                    </>
                  )}
                </div>
                {/* The suggestions give way to F2's card and the F4/F8 notices (as AB01 draws them). */}
                {!confirm && !notice && (
                  <div className="xl-coach__sugg">
                    {ctx.suggestions.map((s) => (
                      <button key={s} className="xl-coach__chip" onClick={() => sendTurn(s)} disabled={sending || held}>
                        {s}
                      </button>
                    ))}
                  </div>
                )}
              </>
            )}

            {confirm && (
              <AssistConfirmCard
                problemId={confirm.problemId ?? ctx.problemId}
                problemTitle={ctx.problemTitle}
                onAsk={askConfirmed}
                onCancel={cancelConfirm}
              />
            )}

            {notice && <CoachNoticeNote notice={notice} onElapsed={() => setNotice(null)} onRetry={retryClosed} />}

            {failure && (
              <CoachFailureNote
                reason={failure.reason}
                model={coachDefault?.model ?? key.default_model}
                provider={key.provider}
                onPicked={() => setFailure(null)}
                onRetry={retryProvider}
              />
            )}
          </>
        )}
      </div>

      <div className="xl-coach__foot">
        {usable ? (
          <>
            <div className="xl-coach__input">
              <input
                ref={inputRef}
                placeholder={paused ? "Paused while a mock or touch is live" : notice?.kind === "daily" ? `Back at ${notice.at} your time` : "Ask your coach…"}
                aria-label="Message coach"
                // Paused and capped for the day show the board's reason as the placeholder;
                // the draft is kept and comes back when the hold clears.
                value={paused || dailyCapped ? "" : input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" && !e.shiftKey) {
                    e.preventDefault();
                    submit();
                  }
                }}
                disabled={inputDisabled}
              />
              <button
                className="ds-iconbtn xl-coach__send"
                aria-label="Send"
                aria-disabled={held || undefined}
                onClick={submit}
                disabled={!held && (sending || input.trim() === "")}
              >
                <Icon name="arrow" />
              </button>
            </div>
            {(notice?.kind === "assist_unavailable" || notice?.kind === "coach_state_unavailable") && (
              <div className="xl-coach__foot-meta">
                <Icon name="lock" className="xl-ico--sm" /> Your message wasn't sent. It's still here.
              </div>
            )}
          </>
        ) : (
          <button className="ds-btn ds-btn--primary ds-btn--sm ds-btn--block" onClick={() => navigate("/settings?tab=coach")}>
            <Icon name="key" className="xl-ico--sm" /> Add your key in Settings
          </button>
        )}
      </div>
    </aside>
  );
}

/** ModeChip states the mode the server decided (AB01 F1 / F5 / F6 / F7) in words, not
 *  colour only. It reports; the gateway and coach-prompt@2 enforce. */
function ModeChip({ mode }: { mode: CoachMode }) {
  switch (mode) {
    case "attempt":
      return (
        <span className="ds-badge ds-badge--info">
          <Icon name="lock" className="xl-ico--sm" /> Attempt · hints only — no pattern or solution
        </span>
      );
    case "review":
      return (
        <span className="ds-badge ds-badge--ok">
          <Icon name="check" className="xl-ico--sm" /> Review
        </span>
      );
    case "general":
      return (
        <span className="xl-tag">
          <Icon name="spark" className="xl-ico--sm" /> General
        </span>
      );
    case "locked":
      return (
        <span className="ds-badge ds-badge--err">
          <Icon name="pause" className="xl-ico--sm" /> Paused
        </span>
      );
  }
}

/**
 * AssistConfirmCard is AB01 F2: the D27 honesty prompt, inline in the thread above the
 * composer (not a page modal, so the problem stays visible). Focus moves to Cancel, the
 * least consequential action; Tab moves between the two buttons; Esc is Cancel.
 */
function AssistConfirmCard({
  problemId,
  problemTitle,
  onAsk,
  onCancel,
}: {
  problemId?: string;
  problemTitle?: string;
  onAsk: () => void;
  onCancel: () => void;
}) {
  const askRef = useRef<HTMLButtonElement>(null);
  const cancelRef = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  useEffect(() => cancelRef.current?.focus(), []);
  const which = [problemId ? `#${problemId}` : "", problemTitle ?? ""].filter(Boolean).join(" ") || "this problem";

  function onKeyDown(e: ReactKeyboardEvent<HTMLElement>) {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      onCancel();
    } else if (e.key === "Tab") {
      e.preventDefault();
      (document.activeElement === askRef.current ? cancelRef : askRef).current?.focus();
    }
  }

  return (
    <section className="xl-coach__confirm" role="alertdialog" aria-labelledby={`${titleId}-t`} aria-describedby={`${titleId}-d`} onKeyDown={onKeyDown}>
      <h3 id={`${titleId}-t`}>
        <Icon name="alert" className="xl-ico--sm" /> Ask the coach about this problem?
      </h3>
      <p id={`${titleId}-d`}>
        Using any AI during a counted attempt caps it at <b>Assisted</b>.
      </p>
      <p>Only this attempt on {which} is affected.</p>
      <div className="xl-coach__confirm-act">
        <button ref={askRef} type="button" className="ds-btn ds-btn--primary ds-btn--sm" onClick={onAsk}>
          Ask — cap at Assisted
        </button>
        <button ref={cancelRef} type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </section>
  );
}

/** CoachNoticeNote renders a turn the server refused before sending (AB01 F4, F8). */
function CoachNoticeNote({ notice, onElapsed, onRetry }: { notice: CoachNotice; onElapsed: () => void; onRetry: (t: PendingTurn) => void }) {
  switch (notice.kind) {
    case "assist_unavailable":
    case "coach_state_unavailable":
      return (
        <div className="xl-coach__note xl-coach__note--err" role="alert">
          <Icon name="alert" className="xl-ico--sm" />
          <div className="xl-coach__note-txt">
            {notice.kind === "assist_unavailable"
              ? "We couldn't record coach use on your attempt, so nothing was sent. Try again."
              : "We couldn't check your practice state, so nothing was sent. Try again in a moment."}
            <div className="xl-coach__note-act">
              <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => onRetry(notice.retry)}>
                <Icon name="refresh" className="xl-ico--sm" /> Try again
              </button>
            </div>
          </div>
        </div>
      );
    case "rate":
      return <RateLimitedNote until={notice.until} onElapsed={onElapsed} />;
    case "busy":
      return <HeldNote until={notice.until} onElapsed={onElapsed} text="One reply at a time — wait for the current reply" />;
    case "daily":
      return (
        <HeldNote
          until={notice.until}
          onElapsed={onElapsed}
          text={`You've reached today's 300-message limit on your key. It resets at 00:00 UTC (${notice.at} your time).`}
        />
      );
  }
}

/** RateLimitedNote is F8a: the seconds count down from Retry-After and the notice clears
 *  at 0. The region is announced once; the number itself is aria-live="off", so the
 *  countdown updates what is seen without re-announcing every second. */
function RateLimitedNote({ until, onElapsed }: { until: number; onElapsed: () => void }) {
  const left = useSecondsLeft(until, onElapsed);
  return (
    <div className="xl-coach__note" role="status">
      <Icon name="clock" className="xl-ico--sm" />
      <div className="xl-coach__note-txt">
        You're sending quickly — try again in{" "}
        <span className="xl-coach__mono" aria-live="off">
          {left} s
        </span>
      </div>
    </div>
  );
}

/** HeldNote is F8b / F8c: a fixed line that clears itself once Retry-After has passed. */
function HeldNote({ until, onElapsed, text }: { until: number; onElapsed: () => void; text: string }) {
  const elapsed = useRef(onElapsed);
  elapsed.current = onElapsed;
  useEffect(() => {
    const t = window.setTimeout(() => elapsed.current(), Math.max(0, until - Date.now()));
    return () => window.clearTimeout(t);
  }, [until]);
  return (
    <div className="xl-coach__note" role="status">
      <Icon name="clock" className="xl-ico--sm" />
      <div className="xl-coach__note-txt">{text}</div>
    </div>
  );
}

/** useSecondsLeft counts the whole seconds left until `until` (epoch ms), calling
 *  onElapsed once it reaches 0. It re-reads the clock each tick rather than counting
 *  ticks, so a throttled background tab still clears on time. */
function useSecondsLeft(until: number, onElapsed: () => void): number {
  const left = () => Math.max(0, Math.ceil((until - Date.now()) / 1000));
  const [secs, setSecs] = useState(left);
  const elapsed = useRef(onElapsed);
  elapsed.current = onElapsed;
  useEffect(() => {
    const iv = window.setInterval(() => {
      const s = Math.max(0, Math.ceil((until - Date.now()) / 1000));
      setSecs(s);
      if (s === 0) {
        window.clearInterval(iv);
        elapsed.current();
      }
    }, 1000);
    return () => window.clearInterval(iv);
  }, [until]);
  return secs;
}

/** secondsToUtcMidnight is the fallback when a daily-cap 429 carries no Retry-After. */
function secondsToUtcMidnight(now: number): number {
  const d = new Date(now);
  return Math.ceil((Date.UTC(d.getUTCFullYear(), d.getUTCMonth(), d.getUTCDate() + 1) - now) / 1000);
}

/** localClock renders an instant as a clock time in the browser's locale ("5:30 AM"). */
function localClock(at: number): string {
  return new Date(at).toLocaleTimeString([], { hour: "numeric", minute: "2-digit" });
}

/**
 * The line each provider refusal gets, copied verbatim from the frozen AB01 (m1-10 task 4).
 * A reason absent from this map (including "auth", which routes to Settings instead, and
 * "unavailable") falls through to the generic in-reply note.
 *
 * Why one line each: v1 reported every non-auth refusal as one "out of credit or limited",
 * so a learner barred from a model, or in an unsupported region, was told to top up a fully
 * funded account. Each line now names what actually happened and what to do about it.
 */
const FAILURE_LINES: Partial<Record<CoachErrorReason, string>> = {
  // F10 — quota, billing, spend caps and rate limits share one line: all of them mean
  // "the account, not the key", and all of them are fixed by topping up or waiting.
  quota: "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry.",
  rate_limit: "Your provider says this key is out of credit or rate-limited. Top up or wait, then retry.",
  // F10's variant strip.
  region: "Your provider doesn't serve this region.",
};

/** hasFailureNote reports whether AB01 gives this reason a note of its own. `model_access`
 *  has a whole frame (F11) rather than a fixed line, so it isn't in FAILURE_LINES. */
function hasFailureNote(reason: CoachErrorReason): boolean {
  return reason === "model_access" || reason in FAILURE_LINES;
}

/**
 * CoachFailureNote is the alert for a provider refusal the board gives its own line
 * (AB01 F10, F10's region variant, F11). The key stays enabled throughout — only an auth
 * failure disables one — so the composer stays usable and the learner can retry; F10 says
 * so beside the notice ("Your Anthropic key is still on.").
 *
 * For `model_access` it also carries F11's affordance: the model the key may not use stays
 * listed and marked in the catalog switcher, and picking another one sets the `coach`
 * feature default and closes the list.
 */
function CoachFailureNote({
  reason,
  model,
  provider,
  onPicked,
  onRetry,
}: {
  reason: CoachErrorReason;
  model: string;
  provider: string;
  onPicked: () => void;
  onRetry: () => void;
}) {
  const catalog = useCoachModels();
  const put = usePutCoachKey();
  const [open, setOpen] = useState(reason === "model_access");
  const pickRef = useRef<HTMLButtonElement>(null);

  if (reason === "region") {
    // F10's variant: the line alone — retrying from the same region can't help.
    return (
      <div className="xl-coach__note" role="alert">
        <Icon name="alert" className="xl-ico--sm" />
        <div className="xl-coach__note-txt">{FAILURE_LINES.region}</div>
      </div>
    );
  }

  if (reason !== "model_access") {
    return (
      <>
        <div className="xl-coach__note" role="alert">
          <Icon name="alert" className="xl-ico--sm" />
          <div className="xl-coach__note-txt">
            {FAILURE_LINES[reason]}
            <div className="xl-coach__note-act">
              <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={onRetry}>
                <Icon name="refresh" className="xl-ico--sm" /> Retry
              </button>
            </div>
          </div>
        </div>
        <div className="xl-coach__keyline">
          <span className="ds-dot ds-dot--ok" /> Your {providerLabel(provider)} key is still on.
        </div>
      </>
    );
  }

  return (
    <div className="xl-coach__note xl-coach__note--err" role="alert">
      <Icon name="alert" className="xl-ico--sm" />
      <div className="xl-coach__note-txt">
        <span>
          This key can't use <span className="ds-mono">{model}</span>. Pick another model.
        </span>
        <div className="xl-coach__note-act">
          <button ref={pickRef} type="button" className="ds-btn ds-btn--secondary ds-btn--sm" aria-haspopup="listbox" aria-expanded={open} onClick={() => setOpen((o) => !o)}>
            Pick another model <Icon name="chevdown" className="xl-ico--sm" />
          </button>
        </div>
        {open && (
          <div
            onKeyDown={(e) => {
              // F11: Esc closes the list back to its button (and not the whole panel).
              if (e.key === "Escape") {
                e.preventDefault();
                setOpen(false);
                pickRef.current?.focus();
              }
            }}
          >
            <CoachModelMenu
              groups={[{ provider, models: catalogModelsFor(catalog.data?.models, provider, model) }]}
              asOf={catalog.data?.as_of}
              selected={model}
              customProvider={provider}
              failing={model}
              onPick={(p, picked) => {
                if (picked !== model) put.mutate({ provider: p, default: true, feature: "coach", default_model: picked });
                setOpen(false);
                onPicked();
              }}
              onManage={() => setOpen(false)}
            />
          </div>
        )}
      </div>
    </div>
  );
}

/** CoachEmptyState prompts the user to add (or re-enable) a provider key, routing to
 *  Settings — the coach is off with no usable key (AB01 F15, v1 parity). Focus lands on
 *  Open Settings, which is the one thing there is to do here. */
function CoachEmptyState({ disabled, onOpenSettings }: { disabled: boolean; onOpenSettings: () => void }) {
  const openSettings = useRef<HTMLButtonElement>(null);
  useEffect(() => openSettings.current?.focus(), []);
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, textAlign: "center", padding: "18px 6px" }}>
      <Icon name="key" style={{ color: "var(--ds-teal)", margin: "0 auto" }} />
      <div style={{ fontSize: 13.5, fontWeight: 600 }}>{disabled ? "Your coach key is turned off" : "Your AI coach is off"}</div>
      <div style={{ fontSize: 12.5, color: "var(--ds-muted)", lineHeight: 1.5 }}>
        {disabled
          ? "Re-enable your provider key in Settings to turn the coach back on."
          : "Add your own OpenAI or Anthropic API key to enable Socratic hints during attempts and a reviewer after each solve."}
      </div>
      <button ref={openSettings} className="ds-btn ds-btn--secondary ds-btn--sm" style={{ alignSelf: "center" }} onClick={onOpenSettings}>
        <Icon name="settings" className="xl-ico--sm" /> Open Settings
      </button>
    </div>
  );
}

// --- page context ---

/** CoachContext is the per-page context the panel shows + sends to the coach. */
interface CoachContext {
  context: string; // stable thread key, e.g. "problem:16"
  kind: string; // problem | concept | dashboard | ...
  label: string; // chip display, e.g. "Problem 16 · 3Sum"
  short: string; // a short noun for the intro line, e.g. "3Sum" / "this screen"
  chipIcon: IconName;
  suggestions: string[];
  problemId?: string;
  problemTitle?: string;
  pattern?: string;
  stage?: string;
  recentOutcome?: string;
}

/** Minimal shape read from the cached Problem aggregate to enrich the chip. */
interface CachedProblem {
  problem?: { title?: string; pattern?: string };
  state?: { stageReached?: string; status?: string; lastOutcome?: string | null };
}

/**
 * useCoachContext derives the page context from the route (+ the cached Problem
 * aggregate for the richer Problem chip). It uses the pathname and useCourse(), not route
 * params, because the panel is mounted in the layout above the routed screen.
 *
 * The context is the coach thread key (coach_thread UNIQUE (account_id, page_context)),
 * so every course-scoped context carries its course (m1-03, t0 §7): `<course>:concept:<slug>`,
 * `<course>:week:<n>`, `<course>:roadmap|dashboard|revision|mistakes|mock|progress` —
 * otherwise week 3 of two courses would share one thread. A problem keeps
 * `problem:<id>` (ids are global), and the account-wide contexts (`catalog`, `settings`,
 * `general`) are unchanged. Course contexts exist only inside an active course: an
 * unknown or coming-soon course's page (NotFound, the teaser) is `general`.
 */
function useCoachContext(pathname: string): CoachContext {
  const qc = useQueryClient();
  const course = useCourse();
  const p = pathname.replace(/\/+$/, "") || "/";
  const rest = course.state === "active" ? courseRest(p, course.slug) : null;
  if (rest === null) return fromGeneric(accountContext(p));

  const problem = rest.match(/^problem\/([^/]+)$/);
  if (problem) {
    const id = decodeURIComponent(problem[1]!);
    // The workspace caches its aggregate per mode (useProblem: ["problem", id, mode]).
    const cached = qc.getQueryData<CachedProblem>(["problem", id, "course"]) ?? qc.getQueryData<CachedProblem>(["problem", id, "practice"]);
    const title = cached?.problem?.title;
    const stage = cached?.state?.stageReached || undefined;
    const parts = [`Problem ${id}`];
    if (title) parts.push(title);
    if (stage) parts.push(stage);
    return {
      context: `problem:${id}`,
      kind: "problem",
      label: parts.join(" · "),
      short: title ?? "this problem",
      chipIcon: "code",
      suggestions: ["Which data structure fits here?", "Am I missing an edge case?", "Nudge me toward the next step"],
      problemId: id,
      problemTitle: title,
      pattern: cached?.problem?.pattern,
      stage,
      recentOutcome: cached?.state?.lastOutcome ?? undefined,
    };
  }

  const concept = rest.match(/^concept\/([^/]+)$/);
  if (concept) {
    const slug = decodeURIComponent(concept[1]!);
    const title = titleFromSlug(slug);
    return {
      context: `${course.slug}:concept:${slug}`,
      kind: "concept",
      label: `Concept — ${title}`,
      short: title,
      chipIcon: "book",
      suggestions: [`Explain the intuition behind ${title}`, "When does this pattern apply?"],
      pattern: title,
    };
  }

  const week = rest.match(/^week\/(\d+)$/);
  if (week) {
    const n = week[1]!;
    return {
      context: `${course.slug}:week:${n}`,
      kind: "week",
      label: `Week ${n}`,
      short: `Week ${n}`,
      chipIcon: "map",
      suggestions: ["What’s the theme this week?", "How should I sequence these problems?"],
    };
  }

  const screen = COURSE_SCREEN_CONTEXT.get(rest);
  if (screen) {
    const [kind, label, icon, short] = screen;
    return fromGeneric([`${course.slug}:${kind}`, label, kind, icon, short]);
  }
  // An unknown sub-route of the course renders NotFound: no course context.
  return fromGeneric(accountContext(p));
}

/** courseRest is the part of path p below the course root ("" at the root itself), or
 *  null when p isn't inside that course. */
function courseRest(p: string, course: string): string | null {
  if (p === `/${course}`) return "";
  return p.startsWith(`/${course}/`) ? p.slice(course.length + 2) : null;
}

/** A generic context: [context key, chip label, kind, chip icon, intro noun]. */
type GenericContext = [string, string, string, IconName, string];

function fromGeneric([context, label, kind, icon, short]: GenericContext): CoachContext {
  return { context, kind, label, short, chipIcon: icon, suggestions: genericSuggestions(kind) };
}

/** The course screens with a generic context, by the path below the course root:
 *  [kind, chip label, chip icon, intro noun]. The key is `<course>:<kind>`. */
const COURSE_SCREEN_CONTEXT = new Map<string, [string, string, IconName, string]>([
  ["", ["roadmap", "Roadmap", "map", "the roadmap"]],
  ["dashboard", ["dashboard", "Today", "today", "your plan for today"]],
  ["revision", ["revision", "Revision", "refresh", "your revisions"]],
  ["mistakes", ["mistakes", "Mistakes", "journal", "your mistake journal"]],
  ["mock", ["mock", "Mock interview", "target", "mock interviews"]],
  ["progress", ["progress", "Progress", "chart", "your progress"]],
]);

/** accountContext is the context of an account-wide page (not course-scoped). */
function accountContext(p: string): GenericContext {
  switch (p) {
    case "/":
      return ["catalog", "Catalog", "catalog", "grid", "the catalog"];
    case "/settings":
      return ["settings", "Settings", "settings", "settings", "your settings"];
    default:
      return ["general", "xLearn", "general", "spark", "this screen"];
  }
}

function genericSuggestions(kind: string): string[] {
  switch (kind) {
    case "dashboard":
      return ["What should I work on next?", "How am I doing this week?"];
    case "mistakes":
      return ["What’s my biggest weak area?", "How do I stop repeating this mistake?"];
    case "mock":
      return ["How do I structure a 45-minute mock?", "What do interviewers look for?"];
    case "progress":
      return ["Where am I falling behind?", "What should I focus on next?"];
    default:
      return ["What should I work on next?"];
  }
}

/** titleFromSlug turns a kebab slug into a Title Case display name. */
function titleFromSlug(slug: string): string {
  return slug
    .split("-")
    .filter(Boolean)
    .map((w) => w[0]!.toUpperCase() + w.slice(1))
    .join(" ");
}
