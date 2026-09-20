import { Link, useParams, useSearchParams } from "react-router-dom";
import { Icon } from "../components/Icon";
import { InlineMD, Markdown } from "../components/Markdown";
import type { Concept as ConceptData, Problem } from "../lib/curriculum";
import { patternMatchesConcept, useConcept, useWeek } from "../lib/curriculum";

const DIFF_CLASS: Record<Problem["difficulty"], string> = {
  easy: "xl-diff xl-diff--easy",
  med: "xl-diff xl-diff--med",
  hard: "xl-diff xl-diff--hard",
};
const DIFF_SHORT: Record<Problem["difficulty"], string> = { easy: "Easy", med: "Med", hard: "Hard" };

/**
 * Concept (`/xlearn/dsa/concept/:slug`): pattern reading from GET /concepts/:slug —
 * the "when to reach for it" callout (when_to_use_md), the body (body_md via a
 * sanitized Markdown renderer into the `.cn` article styles) and the reusable Go
 * code template. `?week=N` (set when opened from a Week) drives the eyebrow and the
 * "practice this pattern" links; v1 is Go-first, so the C++ tab is stubbed.
 */
export default function Concept() {
  const { slug = "" } = useParams();
  const [params] = useSearchParams();
  const weekParam = params.get("week");
  const weekN = weekParam && /^\d+$/.test(weekParam) ? Number(weekParam) : undefined;
  const concept = useConcept(slug);

  return (
    <div style={{ display: "grid", gridTemplateColumns: "minmax(0,760px) 300px", gap: 34, alignItems: "start", maxWidth: 1120 }}>
      <div className="cn">
        {concept.isLoading && <ArticleSkeleton />}

        {concept.isError && (
          <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
            <Icon name="alert" />
            <span className="xl-mut" style={{ flex: 1 }}>
              {concept.error?.status === 404 ? "This concept doesn’t exist yet." : "Couldn’t load this concept."}
            </span>
            {concept.error?.status !== 404 && (
              <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => concept.refetch()}>
                Retry
              </button>
            )}
          </div>
        )}

        {concept.data && <Article concept={concept.data.concept} weekN={weekN} />}
      </div>

      <RightRail slug={slug} title={concept.data?.concept.title ?? slug} weekN={weekN} />
    </div>
  );
}

function Article({ concept, weekN }: { concept: ConceptData; weekN?: number }) {
  return (
    <>
      <div className="xl-eyebrow">{weekN ? `Pattern · Week ${weekN}` : "Pattern"}</div>
      <h1 style={{ fontSize: 30, fontWeight: 700, letterSpacing: "-.4px", margin: "8px 0 6px" }}>{concept.title}</h1>

      {concept.when_to_use_md && (
        <div className="ds-card ds-card--teal" style={{ padding: "16px 18px", margin: "20px 0" }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 6 }}>
            <Icon name="bulb" className="xl-ico--sm" />
            <span
              className="ds-mono"
              style={{ fontSize: 11, textTransform: "uppercase", letterSpacing: ".6px", color: "var(--ds-teal)" }}
            >
              When should I think about this?
            </span>
          </div>
          <p style={{ margin: 0, color: "var(--ds-text)", fontSize: 13.5, lineHeight: 1.6 }}>
            <InlineMD text={concept.when_to_use_md} />
          </p>
        </div>
      )}

      {concept.body_md && <Markdown source={concept.body_md} />}

      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", margin: "22px 0 10px" }}>
        <h2 style={{ margin: 0 }}>Reusable template</h2>
        <div className="ds-seg" role="group" aria-label="Template language">
          <button type="button" className="ds-seg__btn ds-seg__btn--on" aria-pressed="true">
            Go
          </button>
          <button type="button" className="ds-seg__btn" disabled title="Go-first curriculum — a C++ template comes later">
            C++
          </button>
        </div>
      </div>
      {concept.code_template ? (
        <pre className="xl-code">{concept.code_template}</pre>
      ) : (
        <p style={{ fontSize: 12.5, color: "var(--ds-muted)" }}>No code template for this concept yet.</p>
      )}
      <p style={{ fontSize: 12.5, color: "var(--ds-muted)", marginTop: 8 }}>
        v1 is Go-first — the template above is the reusable shape; swap the state and the loop condition per problem.
      </p>
    </>
  );
}

// Right rail: real problems that use this pattern, sourced from the week the concept
// was opened from (matched on the problem's pattern) — never fabricated links. The
// "Practice now" button routes into the guided Problem workspace (S05 stub).
function RightRail({ slug, title, weekN }: { slug: string; title: string; weekN?: number }) {
  const week = useWeek("dsa", weekN ?? 0);
  const related =
    weekN && week.data ? week.data.problems.filter((p) => patternMatchesConcept(p.pattern, slug) || patternMatchesConcept(p.pattern, title)) : [];
  const practiceTo = related[0]
    ? `/dsa/problem/${related[0].id}`
    : weekN
      ? `/dsa/week/${weekN}`
      : "/dsa";

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 16, position: "sticky", top: 0 }}>
      <div className="xl-panel">
        <div className="xl-panel__h" style={{ padding: "12px 14px" }}>
          <Icon name="code" className="xl-ico--sm" />
          <h3 style={{ fontSize: 13 }}>Practice this pattern</h3>
        </div>
        <div style={{ padding: 6 }}>
          {related.length > 0 ? (
            related.map((p) => (
              <Link
                key={p.id}
                to={`/dsa/problem/${p.id}`}
                style={{ display: "flex", alignItems: "center", gap: 9, padding: "9px 10px", borderRadius: 7, textDecoration: "none", color: "inherit" }}
              >
                <span className="ds-mono" style={{ fontSize: 11, color: "var(--ds-muted)" }}>
                  #{p.id}
                </span>
                <span style={{ flex: 1, fontSize: 12.5 }}>{p.title}</span>
                <span className={DIFF_CLASS[p.difficulty]}>{DIFF_SHORT[p.difficulty]}</span>
              </Link>
            ))
          ) : (
            <div className="xl-mut" style={{ padding: "6px 10px", fontSize: 12, color: "var(--ds-muted)" }}>
              {weekN ? "See this week’s problems to practice." : "Open this concept from a week to see its problems."}
            </div>
          )}
        </div>
      </div>

      <Link className="ds-btn ds-btn--primary ds-btn--block" to={practiceTo}>
        <Icon name="play" className="xl-ico--sm" /> Practice now
      </Link>
    </div>
  );
}

function ArticleSkeleton() {
  return (
    <>
      <div className="xl-skel" style={{ height: 16, width: 120 }} />
      <div className="xl-skel" style={{ height: 34, width: "60%", margin: "12px 0 18px" }} />
      <div className="xl-skel" style={{ height: 96, marginBottom: 22 }} />
      <div className="xl-skel" style={{ height: 14, width: "90%", marginBottom: 8 }} />
      <div className="xl-skel" style={{ height: 14, width: "85%", marginBottom: 8 }} />
      <div className="xl-skel" style={{ height: 14, width: "70%", marginBottom: 22 }} />
      <div className="xl-skel" style={{ height: 160 }} />
    </>
  );
}
