import { useEffect, useRef, useState } from "react";
import { Link } from "react-router-dom";
import { usePaths, type Path } from "../lib/curriculum";
import { Icon } from "./Icon";

/** A short mono code for a path's badge (DSA, SYS, …); falls back to the slug head. */
const SHORT_CODE: Record<string, string> = {
  dsa: "DSA",
  "system-design": "SYS",
  "go-concurrency": "GO",
  "lld-ood": "OOD",
  sql: "SQL",
  behavioral: "STAR",
};
function shortCode(slug: string): string {
  return SHORT_CODE[slug] ?? slug.replace(/[^a-z0-9]/gi, "").slice(0, 3).toUpperCase();
}

/**
 * PathSwitcher is the top-bar curriculum selector (F001): it replaces the removed ⌘K
 * search and is shown only inside a curriculum. It names the current path and opens a
 * menu to jump between started/available paths or back to the Catalog ("Browse all
 * paths"). Coming-soon paths are listed but locked. Accessible button-disclosure
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
  const active = all.filter((p) => p.status === "active");
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
        <span className="xl-pathsw__ic">{shortCode(slug)}</span>
        <span className="xl-pathsw__t">{title}</span>
        <Icon name="chevdown" className="xl-ico--sm" />
      </button>

      {open && (
        <div className="xl-topsw__menu" role="menu">
          {active.map((p) => (
            <Link
              key={p.slug}
              to={`/${p.slug}/dashboard`}
              role="menuitem"
              onClick={close}
              className={
                p.slug === slug ? "xl-pathsw__opt xl-pathsw__opt--active" : "xl-pathsw__opt"
              }
            >
              <span className="ds-dot ds-dot--ok" /> {p.title}
            </Link>
          ))}
          {comingSoon.map((p: Path) => (
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
