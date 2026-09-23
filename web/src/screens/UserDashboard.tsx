import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Icon, IconSprite, type IconName } from "../components/Icon";
import { CompletionByPhase, HeatmapGrid, PatternMasteryPanel } from "../components/ProgressViews";
import { Topbar } from "../components/Topbar";
import { useMe } from "../lib/auth";
import { usePublicProfile } from "../lib/profile";
import type { PublicProfile, PublicProfileCourse } from "../lib/profile";
import type { HeatmapDay } from "../lib/progress";

/**
 * UserDashboard (`/xlearn/u/<username>`, F009 / ADR-0024, ADR-0025): the PUBLIC, no-login profile —
 * xLearn's LeetCode-style page. Rendered OUTSIDE AuthedShell (like /auth), it reads the
 * public aggregation GET /u/{username} (non-PII only). Layout (F009 review): a 20:80 grid —
 * a left identity column (silhouette, name, @handle, region, join date, activity heatmap) and
 * a right column (solved / streak / mock tiles + collapsible per-course stats). The header
 * mirrors the authenticated app header when the VIEWER is signed in, and only offers "Sign in"
 * to anonymous visitors. It renders its own IconSprite (no app shell here).
 */
export default function UserDashboard() {
  const { username = "" } = useParams();
  const q = usePublicProfile(username);

  return (
    <div style={{ minHeight: "100vh", background: "var(--ds-bg)", color: "var(--ds-text)" }}>
      <IconSprite />
      <PublicHeader />
      <main style={{ width: "100%", maxWidth: 1200, margin: "0 auto", padding: "28px 20px 80px" }}>
        {q.isLoading && <ProfileSkeleton />}
        {q.isError &&
          (q.error.status === 404 ? (
            <NotFoundProfile username={username} />
          ) : (
            <div className="xl-panel" style={{ padding: 20, display: "flex", alignItems: "center", gap: 12 }}>
              <Icon name="alert" />
              <span style={{ flex: 1, color: "var(--ds-dim)" }}>Couldn’t load this profile.</span>
              <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => q.refetch()}>
                Retry
              </button>
            </div>
          ))}
        {q.data && <ProfileBody data={q.data} />}
      </main>
    </div>
  );
}

/**
 * The top bar. When the VIEWER is signed in, reuse the app's plain header (brand + coach +
 * bell + account menu) so a public profile feels consistent with the rest of the app. When
 * anonymous, show the brand + a Sign-in link. While /me is still loading, show the brand only
 * (no flash of "Sign in" before we know the viewer is authenticated).
 */
function PublicHeader() {
  const me = useMe();
  if (me.data) return <Topbar variant="plain" />;
  return (
    <div className="xl-topbar">
      <BrandLink />
      <div className="xl-topbar__spacer" />
      {!me.isLoading && (
        <Link to="/auth" className="ds-btn ds-btn--secondary ds-btn--sm">
          Sign in
        </Link>
      )}
    </div>
  );
}

function BrandLink() {
  return (
    <Link to="/" className="xl-topbar__brand" aria-label="xLearn — home">
      <span className="xl-brand__mark" style={{ width: 26, height: 26, fontSize: 14 }}>
        x
      </span>
      <span className="xl-brand__name">
        x<b>Learn</b>
      </span>
    </Link>
  );
}

function ProfileBody({ data }: { data: PublicProfile }) {
  return (
    <div className="xl-pubgrid">
      <aside className="xl-pubside">
        <ProfileCard user={data.user} />
        <ActivityCard days={data.heatmap?.days ?? []} />
      </aside>
      <section style={{ minWidth: 0, display: "flex", flexDirection: "column", gap: 20 }}>
        <TotalsRow data={data} />
        <CoursesList courses={data.courses} />
      </section>
    </div>
  );
}

// --- left column: identity + activity ---

/** The identity card: silhouette avatar, display name, @username, region, join date. */
function ProfileCard({ user }: { user: PublicProfile["user"] }) {
  const joined = formatJoined(user.joinedAt);
  return (
    <div className="xl-pubcard">
      <span className="ds-avatar" style={{ width: 88, height: 88 }} aria-label={`${user.displayName} avatar`}>
        <Silhouette />
      </span>
      <h1 style={{ margin: "14px 0 0", fontSize: 21, fontWeight: 700, letterSpacing: "-.3px" }}>{user.displayName}</h1>
      <div className="ds-mono" style={{ marginTop: 4, fontSize: 13, color: "var(--ds-teal)" }}>@{user.username}</div>
      <div style={{ marginTop: 16, display: "flex", flexDirection: "column", gap: 9, fontSize: 12.5, color: "var(--ds-dim)" }}>
        {user.region && <MetaRow icon="clock" text={user.region} />}
        {joined && <MetaRow icon="cal" text={`Joined ${joined}`} />}
      </div>
    </div>
  );
}

function MetaRow({ icon, text }: { icon: IconName; text: string }) {
  return (
    <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
      <span style={{ color: "var(--ds-muted)" }}>
        <Icon name={icon} className="xl-ico--sm" />
      </span>
      {text}
    </span>
  );
}

/** The 16-week activity heatmap, compact for the narrow left column (F009 review). */
function ActivityCard({ days }: { days: HeatmapDay[] }) {
  return (
    <div className="xl-pubcard">
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 12 }}>
        <span style={{ color: "var(--ds-teal)" }}>
          <Icon name="cal" className="xl-ico--sm" />
        </span>
        <b style={{ fontSize: 13 }}>Activity</b>
        <span style={{ marginLeft: "auto", fontSize: 10.5, color: "var(--ds-muted)" }}>16 wks · solves + reviews</span>
      </div>
      <div style={{ overflowX: "auto" }}>
        <HeatmapGrid days={days} weeks={16} cell={10} legend="below" />
      </div>
    </div>
  );
}

// --- right column: totals + courses ---

/** The account-wide totals row: problems solved, current streak, mock best. */
function TotalsRow({ data }: { data: PublicProfile }) {
  const { totals, mock } = data;
  return (
    <div className="xl-pubstats">
      <div className="xl-stat">
        <div className="xl-stat__l">Problems solved</div>
        <div className="xl-stat__v">{totals.solved}</div>
      </div>
      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-warn)" }}>
            <Icon name="flame" className="xl-ico--sm" />
          </span>{" "}
          Current streak
        </div>
        <div className="xl-stat__v" style={{ color: "var(--ds-warn)" }}>
          {totals.streak.current}
          <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> days</span>
        </div>
        <div className="xl-stat__d">Longest: {totals.streak.longest} days</div>
      </div>
      <div className="xl-stat">
        <div className="xl-stat__l">
          <span style={{ color: "var(--ds-violet)" }}>
            <Icon name="target" className="xl-ico--sm" />
          </span>{" "}
          Mock best
        </div>
        {mock.count === 0 ? (
          <>
            <div className="xl-stat__v">
              —<span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / 35</span>
            </div>
            <div className="xl-stat__d">No mocks yet</div>
          </>
        ) : (
          <>
            <div className="xl-stat__v">
              {mock.best}
              <span style={{ fontSize: 13, color: "var(--ds-muted)" }}> / 35</span>
            </div>
            <div className="xl-stat__d">
              Avg {mock.average} · {mock.count} mock{mock.count === 1 ? "" : "s"}
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function CoursesList({ courses }: { courses: PublicProfileCourse[] }) {
  if (courses.length === 0) {
    return (
      <div className="xl-pubcard" style={{ color: "var(--ds-muted)", fontSize: 13 }}>
        No course activity yet.
      </div>
    );
  }
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
      {courses.map((c, i) => (
        <CourseRow key={c.slug} course={c} defaultOpen={i === 0} />
      ))}
    </div>
  );
}

/** A collapsible per-course row: collapsed shows the overview (title + solved/total/pct);
 *  expanding reveals the completion-by-phase table + pattern mastery bars. */
function CourseRow({ course, defaultOpen }: { course: PublicProfileCourse; defaultOpen: boolean }) {
  const [open, setOpen] = useState(defaultOpen);
  const bodyId = `course-${course.slug}`;
  return (
    <div className="xl-course">
      <button
        type="button"
        className="xl-course__head"
        aria-expanded={open}
        aria-controls={bodyId}
        onClick={() => setOpen((v) => !v)}
      >
        <span className="xl-course__chev" data-open={open}>
          <Icon name="chevron" className="xl-ico--sm" />
        </span>
        <span style={{ flex: 1, minWidth: 0 }}>
          <span className="xl-eyebrow">Course</span>
          <span style={{ display: "block", fontSize: 15.5, fontWeight: 700, marginTop: 2 }}>{course.title}</span>
        </span>
        <span style={{ width: 160, flex: "none" }}>
          <span className="ds-mono" style={{ display: "block", fontSize: 12, color: "var(--ds-dim)", textAlign: "right", marginBottom: 6 }}>
            {course.solved} / {course.total} · {course.pct}%
          </span>
          {/* The meter lives inside the clickable <button> header, so it must stay phrasing
              content (no <div>). ds-meter/ds-meter__fill rely on block layout (width/height
              don't apply to inline boxes), so force display:block on these spans — otherwise
              the bar never draws. */}
          <span className="ds-meter" style={{ display: "block" }}>
            <span className="ds-meter__fill" style={{ display: "block", width: `${course.pct}%` }} />
          </span>
        </span>
      </button>
      {open && (
        <div id={bodyId} className="xl-course__body">
          <div className="xl-course__grid">
            <CompletionByPhase phases={course.phases} />
            <PatternMasteryPanel patterns={course.patterns} />
          </div>
        </div>
      )}
    </div>
  );
}

// --- bits ---

/** A generic person silhouette placeholder (avatar upload is future work, F009). */
function Silhouette() {
  return (
    <svg viewBox="0 0 24 24" width="52%" height="52%" fill="currentColor" aria-hidden="true">
      <circle cx="12" cy="8.5" r="4.2" />
      <path d="M3.8 20.5a8.2 8.2 0 0 1 16.4 0z" />
    </svg>
  );
}

/** A friendly public 404 when no account has claimed the username. */
function NotFoundProfile({ username }: { username: string }) {
  return (
    <div className="xl-panel" style={{ padding: "48px 24px", textAlign: "center" }}>
      <div style={{ fontSize: 34, marginBottom: 8 }}>🔍</div>
      <h1 style={{ margin: 0, fontSize: 22, fontWeight: 700 }}>No profile for @{username}</h1>
      <p style={{ margin: "10px auto 20px", maxWidth: 420, color: "var(--ds-dim)", fontSize: 13.5 }}>
        This username hasn’t been claimed, or the profile isn’t public. Check the spelling, or explore xLearn.
      </p>
      <Link to="/" className="ds-btn ds-btn--primary">
        Explore xLearn
      </Link>
    </div>
  );
}

function formatJoined(iso: string): string {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return "";
  return new Date(t).toLocaleDateString(undefined, { year: "numeric", month: "short" });
}

function ProfileSkeleton() {
  return (
    <div className="xl-pubgrid">
      <aside className="xl-pubside">
        <div className="xl-pubcard">
          <div className="xl-skel" style={{ width: 88, height: 88, borderRadius: "50%" }} />
          <div className="xl-skel" style={{ height: 20, width: 150, margin: "16px 0 8px" }} />
          <div className="xl-skel" style={{ height: 13, width: 100 }} />
        </div>
        <div className="xl-skel" style={{ height: 150 }} />
      </aside>
      <section style={{ display: "flex", flexDirection: "column", gap: 20 }}>
        <div className="xl-pubstats">
          {[0, 1, 2].map((i) => (
            <div key={i} className="xl-skel" style={{ height: 84 }} />
          ))}
        </div>
        <div className="xl-skel" style={{ height: 64 }} />
      </section>
    </div>
  );
}
