import { useMemo, useState } from "react";
import { Icon } from "../components/Icon";
import {
  MISTAKE_CATEGORIES,
  categoryBadgeClass,
  categoryLabel,
  useMistakes,
  usePatchMistake,
  useWeakArea,
} from "../lib/mistakes";
import type { Mistake, WeakArea } from "../lib/mistakes";

type StatusFilter = "all" | "open" | "closed";

/**
 * Mistakes (`/xlearn/dsa/mistakes`): the 8-category mistake journal (R-MJ1..R-MJ4),
 * backed by the BFF (GET /mistakes + GET /weak-area). Every below-Clean outcome and
 * failed re-solve opens an entry (pattern pre-filled); an entry closes after two clean
 * revisits and re-opens on a later fail — all computed server-side, the client only
 * renders. The weekly weak-area banner flags the top category; the category picker lets
 * the learner classify an auto-opened entry.
 */
export default function Mistakes() {
  const q = useMistakes();
  const weak = useWeakArea();
  const [cat, setCat] = useState<string>("all");
  const [status, setStatus] = useState<StatusFilter>("all");

  const all = useMemo(() => q.data?.mistakes ?? [], [q.data]);
  const closeThreshold = q.data?.closeThreshold ?? 2;

  // Category chip counts are over the WHOLE journal (matching the artboard).
  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const m of all) if (m.category) counts[m.category] = (counts[m.category] ?? 0) + 1;
    return counts;
  }, [all]);

  const rows = useMemo(
    () =>
      all.filter(
        (m) =>
          (cat === "all" || m.category === cat) &&
          (status === "all" || m.status === status),
      ),
    [all, cat, status],
  );

  return (
    <div>
      <div className="xl-page-h" style={{ borderBottom: "1px solid var(--ds-line)", paddingBottom: 16, marginBottom: 20 }}>
        <div>
          <div className="xl-eyebrow">Learn from every miss</div>
          <h1 style={{ marginTop: 6, fontSize: 26, fontWeight: 700, letterSpacing: "-.3px" }}>Mistake journal</h1>
          <p style={{ margin: "8px 0 0", fontSize: 13.5, color: "var(--ds-dim)", maxWidth: 640 }}>
            Every below-Clean outcome lands here. An entry closes after{" "}
            <b style={{ color: "var(--ds-text)" }}>two successful revisits</b>.
          </p>
        </div>
        {q.data && (
          <div style={{ display: "flex", gap: 16 }}>
            <Stat value={q.data.openCount} label="open" color="var(--ds-warn)" />
            <Stat value={q.data.closedCount} label="closed" color="var(--ds-ok)" />
          </div>
        )}
      </div>

      {weak.data && (
        <WeakAreaBanner
          weak={weak.data}
          onShow={(c) => {
            // Filter to the top category's OPEN entries — exactly the set the banner
            // counts, so "Show these N" reveals N rows.
            setCat(c);
            setStatus("open");
          }}
        />
      )}

      {q.isLoading && <JournalSkeleton />}

      {q.isError && (
        <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
          <Icon name="alert" />
          <span className="xl-mut" style={{ flex: 1 }}>Couldn’t load your mistake journal.</span>
          <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
            Retry
          </button>
        </div>
      )}

      {q.data && (
        <>
          <Filters
            cat={cat}
            status={status}
            total={all.length}
            categoryCounts={categoryCounts}
            onCat={setCat}
            onStatus={setStatus}
          />
          <JournalTable rows={rows} closeThreshold={closeThreshold} journalEmpty={all.length === 0} />
          <div style={{ marginTop: 12, fontSize: 11.5, color: "var(--ds-muted)", display: "flex", alignItems: "center", gap: 8 }}>
            <Icon name="refresh" className="xl-ico--sm" />
            A closed entry that fails a future review re-opens automatically and re-enters the revision queue at Day 1.
          </div>
        </>
      )}
    </div>
  );
}

function Stat({ value, label, color }: { value: number; label: string; color: string }) {
  return (
    <div style={{ textAlign: "right" }}>
      <div className="ds-mono" style={{ fontSize: 22, fontWeight: 700, color }}>{value}</div>
      <div style={{ fontSize: 11, color: "var(--ds-muted)" }}>{label}</div>
    </div>
  );
}

function WeakAreaBanner({ weak, onShow }: { weak: WeakArea; onShow: (cat: string) => void }) {
  if (!weak.topCategory) return null;
  // Count the OPEN entries "Show these N" will reveal (it filters to category + open),
  // so the banner number always matches the filtered result. topCount (this week's
  // count) is the weekly signal that PICKED this category; the actionable set is the
  // open entries in it, which the supporting `entries` carry.
  const count = weak.entries.length;
  if (count === 0) return null;
  return (
    <div className="ds-card ds-card--violet" style={{ padding: "16px 20px", marginBottom: 20, display: "flex", alignItems: "center", gap: 16, flexWrap: "wrap" }}>
      <span style={{ width: 40, height: 40, borderRadius: 10, background: "rgba(240,180,41,.14)", display: "grid", placeItems: "center", flex: "none", color: "var(--ds-warn)" }}>
        <Icon name="alert" />
      </span>
      <div style={{ flex: 1, minWidth: 220 }}>
        <div style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", color: "var(--ds-violet)", fontFamily: "var(--ds-font-mono)", marginBottom: 3 }}>
          This week's weak area
        </div>
        <div style={{ fontSize: 15, fontWeight: 600 }}>
          {categoryLabel(weak.topCategory)} — <span style={{ color: "var(--ds-warn)" }}>{count} open {count === 1 ? "entry" : "entries"}</span> to drill
        </div>
        <div style={{ fontSize: 12.5, color: "var(--ds-dim)", marginTop: 3 }}>
          Your most-missed area this week — your next reviews will surface these before you fail them.
        </div>
      </div>
      <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => onShow(weak.topCategory)}>
        <Icon name="filter" className="xl-ico--sm" /> Show these {count}
      </button>
    </div>
  );
}

function Filters({
  cat,
  status,
  total,
  categoryCounts,
  onCat,
  onStatus,
}: {
  cat: string;
  status: StatusFilter;
  total: number;
  categoryCounts: Record<string, number>;
  onCat: (c: string) => void;
  onStatus: (s: StatusFilter) => void;
}) {
  const chips = [{ key: "all", label: "All", count: total }].concat(
    MISTAKE_CATEGORIES.map((c) => ({ key: c.key, label: c.label, count: categoryCounts[c.key] ?? 0 })),
  );
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12, marginBottom: 14, flexWrap: "wrap" }}>
      <div role="group" aria-label="Filter by category" style={{ display: "flex", alignItems: "center", gap: 7, flexWrap: "wrap", flex: 1 }}>
        {chips.map((c) => (
          <button
            key={c.key}
            type="button"
            aria-pressed={cat === c.key}
            className={cat === c.key ? "ds-chip ds-chip--sm ds-chip--warn" : "ds-chip ds-chip--sm"}
            style={{ cursor: "pointer", fontFamily: "inherit" }}
            onClick={() => onCat(c.key)}
          >
            {c.label} <span className="ds-mono" style={{ opacity: 0.7 }}>{c.count}</span>
          </button>
        ))}
      </div>
      <div className="ds-seg" role="group" aria-label="Filter by status">
        <button type="button" aria-pressed={status === "all"} className={status === "all" ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"} onClick={() => onStatus("all")}>All</button>
        <button type="button" aria-pressed={status === "open"} className={status === "open" ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"} onClick={() => onStatus("open")}>Open</button>
        <button type="button" aria-pressed={status === "closed"} className={status === "closed" ? "ds-seg__btn ds-seg__btn--on" : "ds-seg__btn"} onClick={() => onStatus("closed")}>Closed</button>
      </div>
    </div>
  );
}

function JournalTable({
  rows,
  closeThreshold,
  journalEmpty,
}: {
  rows: Mistake[];
  closeThreshold: number;
  journalEmpty: boolean;
}) {
  return (
    <div className="xl-panel" style={{ overflow: "hidden" }}>
      <table className="xl-table">
        <thead>
          <tr>
            <th style={{ width: 150 }}>Problem</th>
            <th style={{ width: 130 }}>Pattern</th>
            <th>My mistake → root cause → correct insight</th>
            <th style={{ width: 190 }}>Category</th>
            <th style={{ width: 96 }}>Revisit</th>
            <th style={{ width: 104 }}>Status</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((m) => (
            <MistakeRow key={m.id} m={m} closeThreshold={closeThreshold} />
          ))}
        </tbody>
      </table>
      {rows.length === 0 && (
        <div style={{ padding: 36, textAlign: "center", color: "var(--ds-muted)", fontSize: 13 }}>
          {journalEmpty
            ? "No mistakes logged yet. A below-clean solve or a failed review opens an entry here automatically."
            : "No entries in this category — clean record here. Keep it that way."}
        </div>
      )}
    </div>
  );
}

function MistakeRow({ m, closeThreshold }: { m: Mistake; closeThreshold: number }) {
  const title = m.problem?.title ?? `Problem ${m.problemId}`;
  return (
    <tr>
      <td>
        <b style={{ display: "block" }}>{title}</b>
        <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-muted)" }}>#{m.problemId}</span>
      </td>
      <td>{m.pattern ? <span className="xl-pat">{m.pattern}</span> : <span style={{ color: "var(--ds-muted)" }}>—</span>}</td>
      <td style={{ maxWidth: 0 }}>
        <MistakeDetail m={m} />
      </td>
      <td>
        <CategoryCell m={m} />
      </td>
      <td className="ds-mono" style={{ fontSize: 11.5, color: "var(--ds-dim)" }}>{revisitLabel(m)}</td>
      <td>
        {m.status === "open" ? (
          <span className="ds-badge ds-badge--warn ds-mono">Open · {Math.min(m.revisitCount, closeThreshold)}/{closeThreshold}</span>
        ) : (
          <span className="ds-badge ds-badge--ok ds-mono">Closed</span>
        )}
      </td>
    </tr>
  );
}

function MistakeDetail({ m }: { m: Mistake }) {
  if (!m.mistake && !m.rootCause && !m.insight) {
    return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>Not yet written up — add your root cause &amp; insight.</span>;
  }
  return (
    <>
      {m.mistake && <div style={{ color: "var(--ds-err)", fontSize: 12 }}>✗ {m.mistake}</div>}
      {m.rootCause && <div style={{ color: "var(--ds-dim)", fontSize: 12, marginTop: 3 }}>↳ {m.rootCause}</div>}
      {m.insight && <div style={{ color: "var(--ds-ok)", fontSize: 12, marginTop: 3 }}>✓ {m.insight}</div>}
    </>
  );
}

// CategoryCell shows the category badge, or the 8-category picker for an uncategorised
// (auto-opened) entry so the learner can classify it (R-MJ2, server-authoritative).
function CategoryCell({ m }: { m: Mistake }) {
  const patch = usePatchMistake();
  if (m.category) {
    return <span className={categoryBadgeClass(m.category)}>{categoryLabel(m.category)}</span>;
  }
  return (
    <select
      aria-label={`Categorise problem ${m.problemId}`}
      className="ds-chip ds-chip--sm"
      defaultValue=""
      disabled={patch.isPending}
      style={{ cursor: "pointer", fontFamily: "inherit", maxWidth: 180 }}
      onChange={(e) => {
        const category = e.target.value;
        if (category) patch.mutate({ id: m.id, patch: { category } });
      }}
    >
      <option value="" disabled>Categorise…</option>
      {MISTAKE_CATEGORIES.map((c) => (
        <option key={c.key} value={c.key}>{c.label}</option>
      ))}
    </select>
  );
}

// revisitLabel renders an entry's next revision as a relative day ("today" / "tomorrow"
// / "in N days"), or "—" for a closed entry, "next review" when unscheduled.
function revisitLabel(m: Mistake): string {
  if (m.status === "closed") return "—";
  if (!m.revisitDate) return "next review";
  const d = new Date(m.revisitDate);
  if (Number.isNaN(d.getTime())) return "soon";
  const dayMs = 24 * 3600_000;
  const startOfDay = (t: Date) => Date.UTC(t.getUTCFullYear(), t.getUTCMonth(), t.getUTCDate());
  const days = Math.round((startOfDay(d) - startOfDay(new Date())) / dayMs);
  if (days <= 0) return "today";
  if (days === 1) return "tomorrow";
  return `in ${days} days`;
}

function JournalSkeleton() {
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div className="xl-skel" style={{ height: 40 }} />
      <div className="xl-skel" style={{ height: 240 }} />
    </div>
  );
}
