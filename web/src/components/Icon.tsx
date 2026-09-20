// Icon system ported from the design-canvas artboards: a single hidden SVG
// sprite of <symbol>s rendered once at the app root, and an <Icon> that
// references one by name via <use>. Stroke/size come from the `.xl-ico` classes
// in theme.css, matching the DS glyph style.

export type IconName =
  | "today"
  | "map"
  | "refresh"
  | "journal"
  | "target"
  | "chart"
  | "book"
  | "cal"
  | "settings"
  | "bell"
  | "search"
  | "check"
  | "close"
  | "chevron"
  | "chevdown"
  | "plus"
  | "minus"
  | "ext"
  | "play"
  | "pause"
  | "clock"
  | "code"
  | "flame"
  | "spark"
  | "lock"
  | "key"
  | "arrow"
  | "alert"
  | "grid"
  | "list"
  | "layers"
  | "crown"
  | "users"
  | "bulb"
  | "flag"
  | "filter"
  | "eye"
  | "mic"
  | "signout"
  | "server"
  | "branch"
  | "db"
  | "chat";

export function Icon({ name, className }: { name: IconName; className?: string }) {
  return (
    <svg className={className ? `xl-ico ${className}` : "xl-ico"} aria-hidden="true">
      <use href={`#i-${name}`} />
    </svg>
  );
}

/** The one-time SVG symbol sprite. Render once, high in the tree. */
export function IconSprite() {
  return (
    <svg aria-hidden="true" style={{ position: "absolute", width: 0, height: 0, overflow: "hidden" }}>
      <symbol id="i-today" viewBox="0 0 24 24">
        <path d="M4 11.5 12 5l8 6.5" />
        <path d="M6 10.2V19h12v-8.8" />
      </symbol>
      <symbol id="i-map" viewBox="0 0 24 24">
        <path d="M9 5 3.5 7v12L9 17l6 2 5.5-2V5L15 7 9 5z" />
        <path d="M9 5v12M15 7v12" />
      </symbol>
      <symbol id="i-refresh" viewBox="0 0 24 24">
        <path d="M21 12a9 9 0 1 1-3-6.7M21 4v4h-4" />
      </symbol>
      <symbol id="i-journal" viewBox="0 0 24 24">
        <rect x="5" y="3" width="14" height="18" rx="2" />
        <path d="M9 3v18M12.5 8h3.5M12.5 12h3.5" />
      </symbol>
      <symbol id="i-target" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="8" />
        <circle cx="12" cy="12" r="3.4" />
        <path d="M12 1.5v3M12 19.5v3M1.5 12h3M19.5 12h3" />
      </symbol>
      <symbol id="i-chart" viewBox="0 0 24 24">
        <path d="M4 19V11M10 19V5M16 19v-6" />
        <path d="M3 21h18" />
      </symbol>
      <symbol id="i-book" viewBox="0 0 24 24">
        <path d="M5 4.5A2 2 0 0 1 7 3h12v15H7a2 2 0 0 0-2 2z" />
        <path d="M5 4.5V19" />
      </symbol>
      <symbol id="i-cal" viewBox="0 0 24 24">
        <rect x="3.5" y="5" width="17" height="16" rx="2" />
        <path d="M3.5 10h17M8 3v4M16 3v4" />
      </symbol>
      <symbol id="i-settings" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="3" />
        <path d="M12 2v3M12 19v3M4.2 4.2l2.1 2.1M17.7 17.7l2.1 2.1M2 12h3M19 12h3M4.2 19.8l2.1-2.1M17.7 6.3l2.1-2.1" />
      </symbol>
      <symbol id="i-bell" viewBox="0 0 24 24">
        <path d="M6 9a6 6 0 0 1 12 0c0 4.5 1.5 5.5 2 6H4c.5-.5 2-1.5 2-6" />
        <path d="M10 20a2 2 0 0 0 4 0" />
      </symbol>
      <symbol id="i-search" viewBox="0 0 24 24">
        <circle cx="11" cy="11" r="7" />
        <path d="M21 21l-4.3-4.3" />
      </symbol>
      <symbol id="i-check" viewBox="0 0 24 24">
        <path d="M4 12l5 5L20 6" />
      </symbol>
      <symbol id="i-close" viewBox="0 0 24 24">
        <path d="M6 6l12 12M18 6 6 18" />
      </symbol>
      <symbol id="i-chevron" viewBox="0 0 24 24">
        <path d="M9 6l6 6-6 6" />
      </symbol>
      <symbol id="i-chevdown" viewBox="0 0 24 24">
        <path d="M6 9l6 6 6-6" />
      </symbol>
      <symbol id="i-plus" viewBox="0 0 24 24">
        <path d="M12 5v14M5 12h14" />
      </symbol>
      <symbol id="i-minus" viewBox="0 0 24 24">
        <path d="M5 12h14" />
      </symbol>
      <symbol id="i-ext" viewBox="0 0 24 24">
        <path d="M7 17 17 7M8 7h9v9" />
      </symbol>
      <symbol id="i-play" viewBox="0 0 24 24">
        <path d="M8 5l12 7-12 7z" />
      </symbol>
      <symbol id="i-pause" viewBox="0 0 24 24">
        <path d="M9 5v14M15 5v14" />
      </symbol>
      <symbol id="i-clock" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="9" />
        <path d="M12 8v4l3 2" />
      </symbol>
      <symbol id="i-code" viewBox="0 0 24 24">
        <path d="M9 8l-4 4 4 4M15 8l4 4-4 4" />
      </symbol>
      <symbol id="i-flame" viewBox="0 0 24 24">
        <path d="M12 3c.5 3 3.5 4 3.5 8a3.5 3.5 0 0 1-7 0c0-1.5.7-2.3 1.2-3 .3 1 .9 1.6 1.5 1.8-.3-2.5-.8-4.8.8-6.8z" />
      </symbol>
      <symbol id="i-spark" viewBox="0 0 24 24">
        <path d="M12 3l1.7 5 5 1.7-5 1.7L12 16l-1.7-4.6-5-1.7 5-1.7z" />
        <path d="M18.5 14.5l.8 2.2 2.2.8-2.2.8-.8 2.2-.8-2.2-2.2-.8 2.2-.8z" />
      </symbol>
      <symbol id="i-lock" viewBox="0 0 24 24">
        <rect x="4.5" y="10.5" width="15" height="9.5" rx="2" />
        <path d="M8 10.5V7a4 4 0 0 1 8 0v3.5" />
      </symbol>
      <symbol id="i-key" viewBox="0 0 24 24">
        <circle cx="8" cy="15" r="4" />
        <path d="M11 12l9-9M18 5l2 2M15 8l2 2" />
      </symbol>
      <symbol id="i-arrow" viewBox="0 0 24 24">
        <path d="M5 12h13M13 6l6 6-6 6" />
      </symbol>
      <symbol id="i-alert" viewBox="0 0 24 24">
        <path d="M12 3 2.5 20h19z" />
        <path d="M12 10v4M12 17h.01" />
      </symbol>
      <symbol id="i-grid" viewBox="0 0 24 24">
        <rect x="3.5" y="3.5" width="7" height="7" rx="1.5" />
        <rect x="13.5" y="3.5" width="7" height="7" rx="1.5" />
        <rect x="3.5" y="13.5" width="7" height="7" rx="1.5" />
        <rect x="13.5" y="13.5" width="7" height="7" rx="1.5" />
      </symbol>
      <symbol id="i-list" viewBox="0 0 24 24">
        <path d="M8 6h13M8 12h13M8 18h13M3.5 6h.01M3.5 12h.01M3.5 18h.01" />
      </symbol>
      <symbol id="i-layers" viewBox="0 0 24 24">
        <path d="M12 3 3 7.5 12 12l9-4.5z" />
        <path d="M3 12.5 12 17l9-4.5M3 17 12 21.5 21 17" />
      </symbol>
      <symbol id="i-crown" viewBox="0 0 24 24">
        <path d="M3 8l4 4 5-7 5 7 4-4v9H3z" />
      </symbol>
      <symbol id="i-users" viewBox="0 0 24 24">
        <circle cx="9" cy="8" r="3.2" />
        <path d="M2.5 20a6.5 6.5 0 0 1 13 0" />
        <path d="M16 5.3a3.2 3.2 0 0 1 0 5.4M18 13.6a6.5 6.5 0 0 1 3.5 6.4" />
      </symbol>
      <symbol id="i-bulb" viewBox="0 0 24 24">
        <path d="M9.5 18h5M10.5 21h3M12 3a6 6 0 0 0-3.8 10.6c.8.7 1.3 1.5 1.3 2.4h5c0-.9.5-1.7 1.3-2.4A6 6 0 0 0 12 3z" />
      </symbol>
      <symbol id="i-flag" viewBox="0 0 24 24">
        <path d="M5 21V4M5 4h11l-2 3.5L16 11H5" />
      </symbol>
      <symbol id="i-filter" viewBox="0 0 24 24">
        <path d="M3 5h18l-7 8v6l-4 2v-8z" />
      </symbol>
      <symbol id="i-eye" viewBox="0 0 24 24">
        <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7z" />
        <circle cx="12" cy="12" r="3" />
      </symbol>
      <symbol id="i-mic" viewBox="0 0 24 24">
        <rect x="9" y="3" width="6" height="11" rx="3" />
        <path d="M5 11a7 7 0 0 0 14 0M12 18v3" />
      </symbol>
      <symbol id="i-signout" viewBox="0 0 24 24">
        <path d="M9 4H5a1 1 0 0 0-1 1v14a1 1 0 0 0 1 1h4" />
        <path d="M16 12H8M13 8l4 4-4 4" />
      </symbol>
      <symbol id="i-server" viewBox="0 0 24 24">
        <rect x="3" y="4" width="18" height="7" rx="2" />
        <rect x="3" y="14" width="18" height="7" rx="2" />
        <path d="M7 7.5h.01M7 17.5h.01" />
      </symbol>
      <symbol id="i-branch" viewBox="0 0 24 24">
        <circle cx="6" cy="5" r="2.4" />
        <circle cx="6" cy="19" r="2.4" />
        <circle cx="18" cy="8" r="2.4" />
        <path d="M6 7.4v9.2M6 12h6a4 4 0 0 0 4-4" />
      </symbol>
      <symbol id="i-db" viewBox="0 0 24 24">
        <ellipse cx="12" cy="6" rx="8" ry="3" />
        <path d="M4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6" />
      </symbol>
      <symbol id="i-chat" viewBox="0 0 24 24">
        <path d="M21 12a8 8 0 0 1-11.5 7.2L4 20l1-4.5A8 8 0 1 1 21 12z" />
      </symbol>
    </svg>
  );
}
