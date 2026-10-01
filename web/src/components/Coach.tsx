import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import type { IconName } from "./Icon";
import { Icon } from "./Icon";
import { useCourse } from "../lib/course";
import {
  CoachChatError,
  catalogModelsFor,
  coachFeatureDefault,
  streamCoachChat,
  useCoachKey,
  useCoachModels,
  useCoachThread,
  usePutCoachKey,
  type CoachChatBody,
  type CoachErrorReason,
  type CoachMessage,
} from "../lib/settings";
import { CoachModelMenu } from "./CoachModelSwitcher";

/**
 * Coach is the persistent AI-coach panel (Problem.dc.html / Concept.dc.html): a FAB that
 * opens a side panel with the current-page context chip, the thread history, and a live
 * chat that streams the reply token-by-token (SSE). It runs on the user's own provider
 * key — with no key it shows an empty state that routes to Settings. The Socratic-vs-
 * reviewer behaviour is decided server-side (the gateway derives it from practice), so
 * this panel only reports context; it never gates the coach itself.
 */
export function Coach() {
  const [open, setOpen] = useState(false);
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

  const [messages, setMessages] = useState<CoachMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
  // The last provider refusal AB01 gives a line of its own (quota, rate limit, region, a
  // model this key may not use). It clears on the next send: it describes one attempt.
  const [failure, setFailure] = useState<CoachErrorReason | null>(null);
  const bodyRef = useRef<HTMLDivElement>(null);

  // Load the server thread when it (re)loads or the page context changes, unless a live
  // exchange is in flight (don't clobber the streaming reply).
  useEffect(() => {
    if (!sending && thread.data) setMessages(thread.data.messages);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [thread.data, ctx.context]);

  // Keep the transcript scrolled to the newest message (scrollTo is absent in jsdom).
  useEffect(() => {
    bodyRef.current?.scrollTo?.({ top: bodyRef.current.scrollHeight });
  }, [messages, open]);

  async function send(text: string) {
    const message = text.trim();
    if (!message || sending) return;
    if (!usable) {
      navigate("/settings?tab=coach");
      return;
    }
    setInput("");
    setSending(true);
    setFailure(null);
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
    };

    try {
      await streamCoachChat(chatBody, appendDelta);
      // Refresh the persisted thread so a later reopen reflects the server's copy.
      qc.invalidateQueries({ queryKey: ["coach-thread", ctx.context] });
    } catch (err) {
      if (err instanceof CoachChatError && err.routesToSettings) {
        // Only an auth failure disables a key, which is what routes back to Settings (F15b).
        qc.invalidateQueries({ queryKey: ["coach-key"] });
        navigate("/settings?tab=coach");
      } else if (err instanceof CoachChatError && hasFailureNote(err.reason)) {
        // A refusal AB01 has a line for: show that line instead of appending to the reply,
        // and drop the empty assistant bubble the send optimistically opened. The key stays
        // enabled — quota, a rate limit, a region and a model this key may not use are all
        // things the learner fixes without re-entering a working key (the v1 bug this
        // taxonomy exists to end).
        setFailure(err.reason);
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
    } finally {
      setSending(false);
    }
  }

  if (!open) {
    return (
      <button className="xl-fab" onClick={() => setOpen(true)} aria-label="Open AI coach">
        <Icon name="spark" className="xl-ico--lg" />
      </button>
    );
  }

  return (
    <aside className="xl-coach" aria-label="AI coach">
      <div className="xl-coach__top">
        <div className="xl-coach__title">
          <Icon name="spark" /> Coach
        </div>
        <span style={{ flex: 1 }} />
        <button className="ds-iconbtn" onClick={() => setOpen(false)} aria-label="Close coach">
          <Icon name="close" />
        </button>
      </div>

      <div className="xl-coach__ctx">
        <Icon name={ctx.chipIcon} className="xl-ico--sm" /> Context · {ctx.label}
      </div>

      <div
        className="xl-coach__body xl-scroll"
        ref={bodyRef}
        role="log"
        aria-live="polite"
        aria-relevant="additions text"
      >
        {keyQuery.isLoading && <div style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>Loading your coach…</div>}

        {!keyQuery.isLoading && !usable && <CoachEmptyState disabled={!!key && !key.enabled} onOpenSettings={() => navigate("/settings?tab=coach")} />}

        {usable && (
          <>
            {messages.map((m, i) => (
              <div
                key={i}
                className={m.role === "user" ? "xl-coach__msg xl-coach__msg--me" : "xl-coach__msg xl-coach__msg--bot"}
                style={{ whiteSpace: "pre-line" }}
              >
                {m.content || (sending && i === messages.length - 1 ? "…" : "")}
              </div>
            ))}

            {messages.length === 0 && (
              <>
                <div className="xl-coach__msg xl-coach__msg--bot">
                  I read the current page and coach you — Socratically while you’re attempting, and as a reviewer once
                  you’ve solved. Ask me anything about {ctx.short}.
                </div>
                <div className="xl-coach__sugg">
                  {ctx.suggestions.map((s) => (
                    <button key={s} className="xl-coach__chip" onClick={() => send(s)} disabled={sending}>
                      {s}
                    </button>
                  ))}
                </div>
              </>
            )}

            {failure && (
              <CoachFailureNote
                reason={failure}
                model={coachDefault?.model ?? key.default_model}
                provider={key.provider}
                onPicked={() => setFailure(null)}
              />
            )}
          </>
        )}
      </div>

      <div className="xl-coach__foot">
        {usable ? (
          <div className="xl-coach__input">
            <input
              placeholder="Ask your coach…"
              aria-label="Message coach"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  void send(input);
                }
              }}
              disabled={sending}
            />
            <button
              className="ds-iconbtn"
              style={{ color: "var(--ds-teal)" }}
              aria-label="Send"
              onClick={() => void send(input)}
              disabled={sending || input.trim() === ""}
            >
              <Icon name="arrow" />
            </button>
          </div>
        ) : (
          <button className="ds-btn ds-btn--primary ds-btn--sm" style={{ width: "100%" }} onClick={() => navigate("/settings?tab=coach")}>
            <Icon name="key" className="xl-ico--sm" /> Add your key in Settings
          </button>
        )}
      </div>
    </aside>
  );
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
 * failure disables one — so the composer stays usable and the learner can retry.
 *
 * For `model_access` it also carries F11's affordance: the model the key may not use stays
 * listed and marked in the catalog switcher, and picking another one sets the `coach`
 * feature default and closes the list.
 */
function CoachFailureNote({ reason, model, provider, onPicked }: { reason: CoachErrorReason; model: string; provider: string; onPicked: () => void }) {
  const catalog = useCoachModels();
  const put = usePutCoachKey();
  const [open, setOpen] = useState(reason === "model_access");

  if (reason !== "model_access") {
    return (
      <div className="xl-coach__note" role="alert">
        <Icon name="alert" className="xl-ico--sm" />
        <span>{FAILURE_LINES[reason]}</span>
      </div>
    );
  }

  return (
    <div className="xl-coach__note xl-coach__note--err" role="alert">
      <Icon name="alert" className="xl-ico--sm" />
      <div style={{ flex: 1, minWidth: 0 }}>
        <span>
          This key can't use <span className="ds-mono">{model}</span>. Pick another model.
        </span>
        <div className="xl-coach__note-act">
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" aria-haspopup="listbox" aria-expanded={open} onClick={() => setOpen((o) => !o)}>
            Pick another model <Icon name="chevdown" className="xl-ico--sm" />
          </button>
        </div>
        {open && (
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
    const cached = qc.getQueryData<CachedProblem>(["problem", id]);
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
