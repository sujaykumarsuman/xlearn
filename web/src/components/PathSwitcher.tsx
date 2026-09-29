import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { coursePath, courseShortCode } from "../lib/course";
import { usePaths } from "../lib/curriculum";
import { Icon } from "./Icon";

/**
 * PathSwitcher is the top-bar course selector (F001, AB02-F3): it replaces the removed ⌘K
 * search and is shown only inside an active course. It names the current course and
 * opens a menu to jump between the open courses (each to its dashboard) or back to the
 * Catalog ("Browse all paths"). Coming-soon courses are listed but locked. All of it
 * comes from the session-gated catalog. Accessible button-disclosure
 * (aria-expanded/haspopup + Esc + outside-click), mirroring the account menu.
 */
export function PathSwitcher({ slug }: { slug: string }) {
  const paths = usePaths();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    const onPointer = (e: PointerEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("pointerdown", onPointer);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("pointerdown", onPointer);
    };
  }, [open]);

  const all = paths.data?.paths ?? [];
  const current = all.find((p) => p.slug === slug);
  // A preview course is listed only for the owner/tester cohort (m1-04); it switches like
  // an active one, marked "Preview" (AB02-F3).
  const openCourses = all.filter((p) => p.status === "active" || p.status === "preview");
  const comingSoon = all.filter((p) => p.status === "coming_soon");
  const title = current?.title ?? slug.toUpperCase();
  const close = () => setOpen(false);

  return (
    <div className="xl-topsw" ref={ref}>
      <button
        type="button"
        className="xl-topsw__btn"
        aria-haspopup="true"
        aria-expanded={open}
        aria-label="Switch curriculum"
        onClick={() => setOpen((v) => !v)}
      >
        <span className="xl-pathsw__ic">{courseShortCode(current ?? { slug })}</span>
        <span className="xl-pathsw__t">{title}</span>
        <Icon name="chevdown" className="xl-ico--sm" />
      </button>

      {open && (
        <div className="xl-topsw__menu" role="menu">
          {openCourses.map((p) => (
            <Link
              key={p.slug}
              to={coursePath(p.slug, "dashboard")}
              role="menuitem"
              onClick={close}
              className={
                p.slug === slug ? "xl-pathsw__opt xl-pathsw__opt--active" : "xl-pathsw__opt"
              }
            >
              <span className="ds-dot ds-dot--ok" /> {p.title}
              {p.status === "preview" && (
                <span className="ds-badge ds-badge--violet" style={{ height: 19, fontSize: 10.5, marginLeft: "auto" }}>
                  Preview
                </span>
              )}
            </Link>
          ))}
          {comingSoon.map((p) => (
            <div key={p.slug} className="xl-pathsw__opt xl-pathsw__opt--locked" aria-disabled="true">
              <Icon name="lock" className="xl-ico--sm" /> {p.title} · <i>Coming soon</i>
            </div>
          ))}
          <div className="xl-acct-sep" />
          <Link className="xl-pathsw__opt" to="/" role="menuitem" onClick={close} style={{ color: "var(--ds-dim)" }}>
            <Icon name="grid" className="xl-ico--sm" /> Browse all paths
          </Link>
        </div>
      )}
    </div>
  );
}
