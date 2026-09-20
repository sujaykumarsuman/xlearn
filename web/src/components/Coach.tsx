import { useState } from "react";
import { useLocation } from "react-router-dom";
import { routeTitle } from "../nav";
import { Icon } from "./Icon";

/**
 * Coach is the persistent AI-coach scaffold: a FAB that opens an empty side
 * panel with a live page-context chip. No LLM wiring this sprint — the coach is
 * connected to the user's key in Sprint 11.
 */
export function Coach() {
  const [open, setOpen] = useState(false);
  const { pathname } = useLocation();
  const context = routeTitle(pathname);

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
        <Icon name="today" className="xl-ico--sm" /> Context · {context}
      </div>

      <div className="xl-coach__body xl-scroll">
        <div className="xl-coach__msg xl-coach__msg--bot">
          Your AI coach reads the current page and helps Socratically during attempts, then reviews
          your code after you solve. It runs on your own API key — wiring arrives in Sprint 11.
        </div>
        <div className="xl-coach__sugg">
          <button className="xl-coach__chip" disabled>
            Explain this screen
          </button>
          <button className="xl-coach__chip" disabled>
            What should I work on next?
          </button>
        </div>
      </div>

      <div className="xl-coach__foot">
        <div className="xl-coach__input">
          <input placeholder="Coach connects in Sprint 11…" aria-label="Message coach" disabled />
          <button className="ds-iconbtn" style={{ color: "var(--ds-teal)" }} aria-label="Send" disabled>
            <Icon name="arrow" />
          </button>
        </div>
      </div>
    </aside>
  );
}
