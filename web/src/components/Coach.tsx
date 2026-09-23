import { useEffect, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import type { IconName } from "./Icon";
import { Icon } from "./Icon";
import {
  CoachChatError,
  streamCoachChat,
  useCoachKey,
  useCoachThread,
  type CoachChatBody,
  type CoachMessage,
} from "../lib/settings";

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
  // The coach answers with the DEFAULT provider's key, so usability tracks that one.
  const key = keyQuery.data?.keys?.find((k) => k.is_default) ?? keyQuery.data?.keys?.[0];
  const usable = !!key && key.enabled;
  const thread = useCoachThread(ctx.context, open && usable);

  const [messages, setMessages] = useState<CoachMessage[]>([]);
  const [input, setInput] = useState("");
  const [sending, setSending] = useState(false);
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
        qc.invalidateQueries({ queryKey: ["coach-key"] });
        navigate("/settings?tab=coach");
      } else {
        setMessages((prev) => {
          const next = prev.slice();
          const last = next[next.length - 1];
          if (last && last.role === "assistant" && last.content === "") {
            next[next.length - 1] = { ...last, content: "⚠️ The coach couldn’t reply just now. Please try again." };
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

/** CoachEmptyState prompts the user to add (or re-enable) a provider key, routing to
 *  Settings — the coach is off with no usable key (sprint-11 acceptance). */
function CoachEmptyState({ disabled, onOpenSettings }: { disabled: boolean; onOpenSettings: () => void }) {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12, textAlign: "center", padding: "18px 6px" }}>
      <Icon name="key" style={{ color: "var(--ds-teal)", margin: "0 auto" }} />
      <div style={{ fontSize: 13.5, fontWeight: 600 }}>{disabled ? "Your coach key is turned off" : "Your AI coach is off"}</div>
      <div style={{ fontSize: 12.5, color: "var(--ds-muted)", lineHeight: 1.5 }}>
        {disabled
          ? "Re-enable your provider key in Settings to turn the coach back on."
          : "Add your own OpenAI or Anthropic API key to enable Socratic hints during attempts and a reviewer after each solve."}
      </div>
      <button className="ds-btn ds-btn--secondary ds-btn--sm" style={{ alignSelf: "center" }} onClick={onOpenSettings}>
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

/** useCoachContext derives the page context from the route (+ the cached Problem
 *  aggregate for the richer Problem chip). It uses the pathname, not route params,
 *  because the panel is mounted in the layout above the routed screen. */
function useCoachContext(pathname: string): CoachContext {
  const qc = useQueryClient();
  const p = pathname.replace(/\/+$/, "") || "/";

  const problem = p.match(/^\/dsa\/problem\/([^/]+)$/);
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

  const concept = p.match(/^\/dsa\/concept\/([^/]+)$/);
  if (concept) {
    const slug = decodeURIComponent(concept[1]!);
    const title = titleFromSlug(slug);
    return {
      context: `concept:${slug}`,
      kind: "concept",
      label: `Concept — ${title}`,
      short: title,
      chipIcon: "book",
      suggestions: [`Explain the intuition behind ${title}`, "When does this pattern apply?"],
      pattern: title,
    };
  }

  const week = p.match(/^\/dsa\/week\/(\d+)$/);
  if (week) {
    const n = week[1]!;
    return {
      context: `week:${n}`,
      kind: "week",
      label: `Week ${n}`,
      short: `Week ${n}`,
      chipIcon: "map",
      suggestions: ["What’s the theme this week?", "How should I sequence these problems?"],
    };
  }

  const [context, label, kind, icon, short] = genericContext(p);
  return {
    context,
    kind,
    label,
    short,
    chipIcon: icon,
    suggestions: genericSuggestions(kind),
  };
}

function genericContext(p: string): [string, string, string, IconName, string] {
  switch (true) {
    case p === "/":
      return ["catalog", "Catalog", "catalog", "grid", "the catalog"];
    case p === "/dsa":
      return ["roadmap", "Roadmap", "roadmap", "map", "the roadmap"];
    case p === "/dsa/dashboard":
      return ["dashboard", "Today", "dashboard", "today", "your plan for today"];
    case p === "/dsa/revision":
      return ["revision", "Revision", "revision", "refresh", "your revisions"];
    case p === "/dsa/mistakes":
      return ["mistakes", "Mistakes", "mistakes", "journal", "your mistake journal"];
    case p === "/dsa/mock":
      return ["mock", "Mock interview", "mock", "target", "mock interviews"];
    case p === "/dsa/progress":
      return ["progress", "Progress", "progress", "chart", "your progress"];
    case p === "/settings":
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
