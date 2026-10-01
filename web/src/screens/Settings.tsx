import { useEffect, useRef, useState, type ReactNode } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import { Icon, type IconName } from "../components/Icon";
import { ProviderLogo } from "../components/ProviderLogo";
import { Spinner } from "../components/States";
import { BudgetFields } from "../components/BudgetFields";
import { type ApiRequestError } from "../lib/api";
import { DEFAULT_WEEKDAY, DEFAULT_WEEKEND, budgetEta, clampWeekday, weekendLabel } from "../lib/budget";
import {
  oauthLinkAction,
  useMe,
  usePatchMe,
  useSetPassword,
  useSetUsername,
  useUnlinkOAuth,
  useUsernameAvailability,
  type Account,
  type WeekendBand,
} from "../lib/auth";
import { CoachModelMenu, type CoachModelGroup } from "../components/CoachModelSwitcher";
import {
  COACH_PROVIDERS,
  catalogModelsFor,
  coachFeatureDefault,
  coachModelLabel,
  coachUsageLine,
  providerLabel,
  useCoachKey,
  useCoachModels,
  useDeleteCoachKey,
  usePopoverDismiss,
  usePutCoachKey,
  type CoachFeature,
  type CoachFeatureDefault,
  type CoachKey,
  type CoachKeyResponse,
  type CoachModelsResponse,
  type CoachUsageMonth,
  type ProviderId,
  type PutCoachKeyBody,
} from "../lib/settings";

/**
 * Settings is the account surface: Profile, Study budget, your Coach and Reminders.
 * Redesigned (F006): a two-column layout — a section rail + a column of soft cards. The rail
 * items are tabs (one section shown at a time), so navigation is a deterministic click rather
 * than a scroll-spy. The Coach section connects one key PER PROVIDER (Anthropic and/or OpenAI)
 * and points each FEATURE (chat coach, interview brain) at its own default model from the
 * server catalog. Fully-rounded (pill) controls + circular tiles. Every write is live
 * (PATCH /me · PUT/DELETE /coach/key).
 */
export default function Settings() {
  const me = useMe();
  const [params] = useSearchParams();
  // Deep links pick the starting tab: ?tab=<id> (e.g. the coach "open settings" prompts), or
  // the account tab when the GitHub link flow returns to ?linked=…/?error=… so its banner shows.
  const [tab, setTab] = useState<string>(() => {
    const t = params.get("tab");
    if (t && RAIL.some((s) => s.id === t)) return t;
    if (params.get("linked") || params.get("error")) return "account";
    return RAIL[0]!.id;
  });

  return (
    <div className="xl-settings">
      <header style={{ marginBottom: 24 }}>
        <div className="xl-eyebrow">Account</div>
        <h1 style={{ fontSize: 28, fontWeight: 800, letterSpacing: "-.4px", marginTop: 8 }}>Settings</h1>
        <p style={{ fontSize: 13.5, color: "var(--ds-dim)", marginTop: 6 }}>
          Your profile, study pace, AI coach and reminders — tuned to how you learn.
        </p>
      </header>

      {me.isLoading && <Centered>Loading your settings…</Centered>}
      {me.isError && (
        <Card icon="alert" title="Couldn’t load your settings">
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <span style={{ flex: 1, color: "var(--ds-dim)", fontSize: 13 }}>Something went wrong reaching your account.</span>
            <button className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => me.refetch()}>
              Retry
            </button>
          </div>
        </Card>
      )}

      {me.data && (
        <div className="xl-settings__grid">
          <SettingsRail account={me.data.account} active={tab} onSelect={setTab} />
          <div className="xl-settings__content" role="tabpanel" id={`panel-${tab}`} aria-labelledby={`tab-${tab}`}>
            {tab === "budget" && <BudgetSection account={me.data.account} />}
            {tab === "coach" && <CoachSection />}
            {tab === "reminders" && <RemindersSection account={me.data.account} />}
            {tab === "account" && <AccountSection account={me.data.account} />}
          </div>
        </div>
      )}
    </div>
  );
}

// --- Sign-in & security (ADR-0023) ---

function AccountSection({ account }: { account: Account }) {
  const hasPassword = !!account.has_password;
  const linked = account.linked_providers ?? [];
  const githubLinked = linked.includes("github");
  // The OAuth link flow redirects back here with ?linked=github or ?error=github_taken.
  const [params, setParams] = useSearchParams();
  const linkedOk = params.get("linked");
  const linkErr = params.get("error");
  const clearBanner = () => {
    const next = new URLSearchParams(params);
    next.delete("linked");
    next.delete("error");
    setParams(next, { replace: true });
  };

  return (
    <Card icon="key" title="Sign-in & security" subtitle="How you sign in to xLearn.">
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        {(linkedOk || linkErr) && (
          <div
            role="status"
            className="xl-set-hint"
            style={{
              justifyContent: "space-between",
              borderColor: linkErr ? "var(--ds-err)" : "rgba(87,211,154,.4)",
              color: linkErr ? "var(--ds-err)" : "var(--ds-ok)",
            }}
          >
            <span>{linkErr ? "That GitHub account is already linked to another xLearn account." : "GitHub connected."}</span>
            <button type="button" className="ds-btn ds-btn--ghost ds-btn--sm" onClick={clearBanner}>
              Dismiss
            </button>
          </div>
        )}
        <UsernameForm account={account} />
        <div style={{ borderTop: "1px solid var(--ds-line)", paddingTop: 16 }}>
          <PasswordForm hasPassword={hasPassword} />
        </div>
        <div style={{ borderTop: "1px solid var(--ds-line)", paddingTop: 16 }}>
          <ProvidersRow githubLinked={githubLinked} canUnlink={hasPassword || linked.length > 1} />
        </div>
      </div>
    </Card>
  );
}

/** UsernameForm claims or changes the account's public handle (F009). Live availability
 *  check (debounced); a change warns that it moves the public dashboard URL. */
function UsernameForm({ account }: { account: Account }) {
  const setU = useSetUsername();
  const current = account.username ?? "";
  const [value, setValue] = useState(current);
  const [debounced, setDebounced] = useState("");

  useEffect(() => {
    const t = setTimeout(() => setDebounced(value.trim().toLowerCase()), 350);
    return () => clearTimeout(t);
  }, [value]);

  const normalized = value.trim().toLowerCase();
  const changed = normalized !== current;
  const avail = useUsernameAvailability(debounced);
  const canSave = changed && normalized.length >= 3 && avail.data?.available === true && !setU.isPending;

  const touch = () => {
    if (setU.isSuccess || setU.isError) setU.reset();
  };
  const save = () => {
    if (!canSave) return;
    setU.mutate(normalized);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div>
        <b style={{ fontSize: 13.5 }}>Username</b>
        <div style={{ fontSize: 11.5, color: "var(--ds-muted)", marginTop: 2 }}>
          Your public dashboard address, and a second way to sign in.{" "}
          {current && (
            <>
              Live at{" "}
              <Link to={`/u/${current}`} className="ds-mono" style={{ color: "var(--ds-teal)" }}>
                /xlearn/u/{current}
              </Link>
              .
            </>
          )}
        </div>
      </div>
      <Field label="Username" htmlFor="username" hint="3–30 chars · lowercase, numbers, hyphens">
        <input
          id="username"
          className="ds-input"
          autoComplete="off"
          autoCapitalize="none"
          spellCheck={false}
          placeholder="your-handle"
          value={value}
          onChange={(e) => {
            touch();
            setValue(e.target.value);
          }}
        />
      </Field>
      <div style={{ display: "flex", alignItems: "center", gap: 12, minHeight: 20 }}>
        <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled={!canSave} onClick={save}>
          {setU.isPending ? "Saving…" : current ? "Change username" : "Claim username"}
        </button>
        <UsernameStatus current={current} normalized={normalized} changed={changed} avail={avail} setU={setU} />
      </div>
      {current && changed && normalized.length >= 3 && (
        <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>
          Changing your username changes your public URL — the old address stops working.
        </span>
      )}
    </div>
  );
}

/** The live validity/availability/result line beneath the username field. */
function UsernameStatus({
  current,
  normalized,
  changed,
  avail,
  setU,
}: {
  current: string;
  normalized: string;
  changed: boolean;
  avail: ReturnType<typeof useUsernameAvailability>;
  setU: ReturnType<typeof useSetUsername>;
}) {
  const ok = (t: string) => (
    <span style={{ fontSize: 12, color: "var(--ds-ok)", display: "inline-flex", alignItems: "center", gap: 5 }}>
      <Icon name="check" className="xl-ico--sm" /> {t}
    </span>
  );
  const err = (t: string) => <span style={{ fontSize: 12, color: "var(--ds-err)" }}>{t}</span>;

  if (setU.isSuccess) return ok("Username updated");
  if (setU.isError) return err(setU.error.message || "Couldn’t save — try again.");
  if (!current && normalized.length === 0) return null;
  if (!changed) return current ? <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>This is your current username.</span> : null;
  if (normalized.length < 3) return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>At least 3 characters.</span>;
  if (avail.isLoading) return <span style={{ fontSize: 12, color: "var(--ds-muted)" }}>Checking…</span>;
  if (avail.data?.available) return ok(`@${normalized} is available`);
  if (avail.data) return err(avail.data.reason ?? "That username isn’t available.");
  return null;
}

function PasswordForm({ hasPassword }: { hasPassword: boolean }) {
  const setPw = useSetPassword();
  const navigate = useNavigate();
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const valid = next.length >= 8 && (!hasPassword || current.length >= 1) && !setPw.isPending;

  const touch = () => {
    if (setPw.isSuccess || setPw.isError) setPw.reset();
  };
  const save = () =>
    setPw.mutate(
      { new_password: next, ...(hasPassword ? { current_password: current } : {}) },
      {
        onSuccess: (res) => {
          setCurrent("");
          setNext("");
          // A password change revokes every session, this one included (m1-04): the hook has
          // dropped the cached account; send the learner to sign in again, with a notice.
          if (res?.reauth) navigate("/auth?notice=password_changed", { replace: true });
        },
      },
    );

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
      <div>
        <b style={{ fontSize: 13.5 }}>{hasPassword ? "Change password" : "Set a password"}</b>
        <div style={{ fontSize: 11.5, color: "var(--ds-muted)", marginTop: 2 }}>
          {hasPassword ? "Update the password you sign in with." : "Add a password so you can sign in without GitHub."}
        </div>
      </div>
      {hasPassword && (
        <Field label="Current password" htmlFor="pw-current">
          <input
            id="pw-current"
            className="ds-input ds-input--mono"
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(e) => {
              touch();
              setCurrent(e.target.value);
            }}
          />
        </Field>
      )}
      <Field label={hasPassword ? "New password" : "Password"} htmlFor="pw-new" hint="at least 8 characters">
        <input
          id="pw-new"
          className="ds-input ds-input--mono"
          type="password"
          autoComplete="new-password"
          value={next}
          onChange={(e) => {
            touch();
            setNext(e.target.value);
          }}
        />
      </Field>
      <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
        <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled={!valid} onClick={save}>
          {setPw.isPending ? "Saving…" : hasPassword ? "Change password" : "Set password"}
        </button>
        {setPw.isSuccess && (
          <span style={{ fontSize: 12, color: "var(--ds-ok)", display: "inline-flex", alignItems: "center", gap: 5 }}>
            <Icon name="check" className="xl-ico--sm" /> Password updated
          </span>
        )}
        {setPw.isError && <span style={{ fontSize: 12, color: "var(--ds-err)" }}>{passwordErrorMessage(setPw.error)}</span>}
      </div>
    </div>
  );
}

function ProvidersRow({ githubLinked, canUnlink }: { githubLinked: boolean; canUnlink: boolean }) {
  const unlink = useUnlinkOAuth();
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      <div>
        <b style={{ fontSize: 13.5 }}>Connected accounts</b>
        <div style={{ fontSize: 11.5, color: "var(--ds-muted)", marginTop: 2 }}>Sign in faster with a linked provider.</div>
      </div>
      <div className="xl-prov" style={{ flexDirection: "row", alignItems: "center", gap: 12, padding: "12px 14px" }}>
        <span className="xl-logo" style={{ width: 36, height: 36, background: "#161b22", border: "1px solid var(--ds-line-2)" }}>
          <GitHubMark />
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 14, fontWeight: 600 }}>GitHub</div>
          <div className="xl-prov__status" style={githubLinked ? undefined : { color: "var(--ds-muted)" }}>
            {githubLinked ? (
              <>
                <span className="xl-dot" /> Connected
              </>
            ) : (
              "Not connected"
            )}
          </div>
        </div>
        {githubLinked ? (
          <button
            type="button"
            className="ds-btn ds-btn--ghost ds-btn--sm"
            style={{ color: canUnlink ? "var(--ds-err)" : "var(--ds-muted)" }}
            disabled={!canUnlink || unlink.isPending}
            title={canUnlink ? undefined : "This is your only way to sign in, so it can’t be disconnected."}
            onClick={() => unlink.mutate("github")}
          >
            {unlink.isPending ? "Disconnecting…" : "Disconnect"}
          </button>
        ) : (
          <form method="post" action={oauthLinkAction("github")}>
            <button type="submit" className="ds-btn ds-btn--secondary ds-btn--sm">
              Connect
            </button>
          </form>
        )}
      </div>
      {githubLinked && !canUnlink && (
        <span style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>
          GitHub is currently your only way to sign in, so it can’t be disconnected. Add a password and both stay
          available — you decide when to remove one.
        </span>
      )}
      {unlink.isError && <span style={{ fontSize: 12, color: "var(--ds-err)" }}>{unlinkErrorMessage(unlink.error)}</span>}
    </div>
  );
}

function passwordErrorMessage(err: ApiRequestError | null): string {
  // identity sheds load with 429 too_many_requests while bcrypt is busy (L3).
  if (err?.code === "too_many_requests" || err?.status === 429) return "Too many attempts right now — try again in a moment.";
  switch (err?.code) {
    case "wrong_password":
      return "Your current password is incorrect.";
    case "weak_password":
      return "Password must be 8–72 characters.";
    default:
      return "Couldn’t update your password. Try again.";
  }
}

function unlinkErrorMessage(err: ApiRequestError | null): string {
  switch (err?.code) {
    case "last_login_method":
      return "Set a password before disconnecting your only sign-in method.";
    default:
      return "Couldn’t disconnect. Try again.";
  }
}

function GitHubMark() {
  return (
    <svg width="18" height="18" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  );
}

// --- Section rail (sticky nav + identity) ---

const RAIL: { id: string; label: string; icon: IconName }[] = [
  { id: "budget", label: "Study budget", icon: "clock" },
  { id: "coach", label: "Your AI coach", icon: "spark" },
  { id: "reminders", label: "Reminders", icon: "bell" },
  { id: "account", label: "Sign-in & security", icon: "key" },
];

function SettingsRail({ account, active, onSelect }: { account: Account; active: string; onSelect: (id: string) => void }) {
  return (
    <aside className="xl-set-rail">
      <ProfileCard account={account} />
      <nav className="xl-set-nav" role="tablist" aria-orientation="vertical" aria-label="Settings sections">
        {RAIL.map((s) => {
          const on = active === s.id;
          return (
            <button
              key={s.id}
              type="button"
              role="tab"
              id={`tab-${s.id}`}
              aria-selected={on}
              aria-controls={`panel-${s.id}`}
              className={on ? "xl-rail xl-rail--on" : "xl-rail"}
              onClick={() => onSelect(s.id)}
            >
              <Icon name={s.icon} className="xl-ico--sm" /> {s.label}
            </button>
          );
        })}
      </nav>
    </aside>
  );
}

/** ProfileCard is the account identity in the rail, styled like a GitHub profile: a large
 *  avatar, the name, an "Edit profile" button, then meta rows (local time from the timezone,
 *  and email). Editing swaps in a compact form (name + timezone); email is read-only. */
function ProfileCard({ account }: { account: Account }) {
  const save = usePatchMe();
  const [editing, setEditing] = useState(false);
  const [name, setName] = useState(account.display_name);
  const [tz, setTz] = useState(account.timezone || "UTC");
  const zones = TIMEZONES.includes(tz) ? TIMEZONES : [tz, ...TIMEZONES];
  const nameValid = name.trim().length > 0;

  const cancel = () => {
    setName(account.display_name);
    setTz(account.timezone || "UTC");
    save.reset();
    setEditing(false);
  };

  const avatar = (
    <div className="xl-pf__avatar">
      <span className="ds-avatar xl-pf__av">{initial(account)}</span>
      <button type="button" className="xl-pf-btn xl-pf__avedit" aria-label="Change avatar" disabled title="Avatar upload is coming soon">
        <PencilIcon />
      </button>
    </div>
  );

  if (editing) {
    return (
      <div className="xl-pf">
        {avatar}
        <div className="xl-pf__form">
          <Field label="Name" htmlFor="pf-name">
            <input
              id="pf-name"
              className={nameValid ? "ds-input" : "ds-input ds-input--invalid"}
              value={name}
              autoFocus
              onChange={(e) => {
                if (save.isError) save.reset();
                setName(e.target.value);
              }}
            />
          </Field>
          <Field label="Timezone" htmlFor="pf-tz">
            <select
              id="pf-tz"
              className="ds-input"
              value={tz}
              onChange={(e) => {
                if (save.isError) save.reset();
                setTz(e.target.value);
              }}
            >
              {zones.map((z) => (
                <option key={z} value={z}>
                  {z}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Email" htmlFor="pf-email" hint="from your sign-in">
            <input id="pf-email" className="ds-input" value={account.email ?? ""} readOnly style={{ color: "var(--ds-dim)" }} />
          </Field>
          <div className="xl-pf__actions">
            <button
              type="button"
              className="ds-btn ds-btn--primary ds-btn--sm"
              disabled={!nameValid || save.isPending}
              onClick={() => save.mutate({ display_name: name.trim(), timezone: tz }, { onSuccess: () => setEditing(false) })}
            >
              {save.isPending ? "Saving…" : "Save"}
            </button>
            <button type="button" className="ds-btn ds-btn--ghost ds-btn--sm" onClick={cancel}>
              Cancel
            </button>
          </div>
          {save.isError && <div className="xl-pf-err">Couldn’t save — try again.</div>}
        </div>
      </div>
    );
  }

  return (
    <div className="xl-pf">
      {avatar}
      <div className="xl-pf__name">{account.display_name || "Learner"}</div>
      <button type="button" className="ds-btn ds-btn--secondary xl-pf__editbtn" onClick={() => setEditing(true)}>
        <PencilIcon /> Edit profile
      </button>
      <div className="xl-pf__meta">
        <div className="xl-pf__metarow">
          <Icon name="clock" className="xl-ico--sm" /> {localTime(account.timezone || "UTC")}
        </div>
        <div className="xl-pf__metarow">
          <MailIcon /> <span className="xl-pf__metaval" title={account.email ?? ""}>{account.email || "—"}</span>
        </div>
      </div>
    </div>
  );
}

/** localTime renders a timezone's wall-clock time + UTC offset, e.g. "15:27 (UTC +05:30)". */
function localTime(tz: string): string {
  try {
    const now = new Date();
    const time = new Intl.DateTimeFormat("en-GB", { timeZone: tz, hour: "2-digit", minute: "2-digit", hour12: false }).format(now);
    const parts = new Intl.DateTimeFormat("en-US", { timeZone: tz, timeZoneName: "longOffset" }).formatToParts(now);
    const off = parts.find((p) => p.type === "timeZoneName")?.value ?? "";
    const utc = off && off !== "GMT" ? off.replace("GMT", "UTC ") : "UTC";
    return `${time} (${utc})`;
  } catch {
    return tz;
  }
}

function PencilIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M4 20h4L18.5 9.5l-4-4L4 16z" />
      <path d="M13.5 6.5l4 4" />
    </svg>
  );
}
function MailIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <rect x="3" y="5" width="18" height="14" rx="2" />
      <path d="M3 7l9 6 9-6" />
    </svg>
  );
}

function initial(account: Account): string {
  const s = (account.display_name || account.email || "?").trim();
  return s ? s[0]!.toUpperCase() : "?";
}

// --- Profile ---

const TIMEZONES = [
  "UTC",
  "Asia/Kolkata",
  "Asia/Singapore",
  "Asia/Tokyo",
  "Asia/Dubai",
  "Europe/London",
  "Europe/Berlin",
  "America/New_York",
  "America/Chicago",
  "America/Los_Angeles",
  "Australia/Sydney",
];

// --- Study budget ---

function BudgetSection({ account }: { account: Account }) {
  const save = usePatchMe();
  const [weekday, setWeekday] = useState(clampWeekday(account.study_budget.weekday_minutes ?? DEFAULT_WEEKDAY));
  const [weekend, setWeekend] = useState<WeekendBand>(account.study_budget.weekend_band ?? DEFAULT_WEEKEND);

  const edit =
    <T,>(setter: (v: T) => void) =>
    (v: T) => {
      if (save.isSuccess || save.isError) save.reset();
      setter(v);
    };

  return (
    <Card icon="clock" title="Study budget" subtitle="xLearn sizes each day’s plan to fit. Reviews always come first.">
      <div style={{ display: "flex", flexDirection: "column", gap: 18 }}>
        <BudgetFields weekday={weekday} weekend={weekend} onWeekday={edit(setWeekday)} onWeekend={edit(setWeekend)} />

        <div className="xl-set-hint">
          <Icon name="bulb" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} />
          <span>
            At {weekday} min/weekday + {weekendLabel(weekend)}/weekend, you’ll finish the 16-week path in ~
            <b style={{ color: "var(--ds-text)" }}>{budgetEta(weekday, weekend)}</b> at your current pace.
          </span>
        </div>

        <SaveRow save={save} onSave={() => save.mutate({ study_budget: { weekday_minutes: weekday, weekend_band: weekend } })} />
      </div>
    </Card>
  );
}

// --- Coach: per-provider keys + per-feature default models (AB01 F13) ---

/**
 * The features that each carry their own default model, in the board's order. `note` is the
 * parenthetical the row shows: only Interview has one, because only Interview constrains
 * which models are offered (interview_brain, t6 §11 — the earlier "realtime-capable" rule
 * was dropped there).
 */
const COACH_FEATURE_ROWS: { feature: CoachFeature; label: string; note?: string }[] = [
  { feature: "coach", label: "Coach" },
  { feature: "interview", label: "Interview", note: "(needs an interview-capable model)" },
];

/** The chip each provider panel wears when it backs a feature's default. */
const COACH_ROLE_TAGS: { feature: CoachFeature; tag: string }[] = [
  { feature: "coach", tag: "Coach default" },
  { feature: "interview", tag: "Interview default" },
];

/**
 * CoachSection is Settings' coach card (AB01 F13). Since m1-10 the per-feature DEFAULTS own
 * the model choice — one default model per feature, from the server catalog — and each
 * provider panel is reduced to what it is actually for: connect, update, remove. v1's
 * per-provider model pills and "Set as default" are gone, because a key is no longer tied
 * to one model and "the default" is no longer a single account-wide thing.
 */
function CoachSection() {
  const coach = useCoachKey();
  const catalog = useCoachModels();
  const keys = coach.data?.keys ?? [];
  const usage = coach.data?.usage_month;
  const coachDefault = coachFeatureDefault(coach.data, "coach");
  // The status badge tracks the key the CHAT coach answers with — a paused interview key
  // doesn't make the coach itself paused.
  const coachKey = keys.find((k) => k.provider === coachDefault?.provider);
  const roles = featureRoleTags(coach.data);

  return (
    <Card
      icon="spark"
      title="Your AI coach (your key)"
      subtitle="Connect one or both providers. Each feature uses its own default model. Keys are stored encrypted."
      right={<CoachStatus coachKey={coachKey} />}
    >
      {coach.isLoading ? (
        <Centered>Loading…</Centered>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div className="xl-set-subh">Default model per feature</div>
          {COACH_FEATURE_ROWS.map((row) => (
            <FeatureDefaultRow key={row.feature} feature={row.feature} label={row.label} note={row.note} keys={keys} current={coachFeatureDefault(coach.data, row.feature)} catalog={catalog.data} />
          ))}
          {usage && <UsageMonthHint usage={usage} />}

          <div className="xl-set-subh">Keys</div>
          {COACH_PROVIDERS.map((p) => {
            const cur = keys.find((k) => k.provider === p.id);
            // Remount on connect/disconnect so the panel's local key-field state re-seeds
            // from the freshly-connected key rather than the empty form.
            return <ProviderPanel key={`${p.id}-${cur ? "on" : "off"}`} providerId={p.id} current={cur} roles={roles[p.id] ?? []} />;
          })}
        </div>
      )}
    </Card>
  );
}

function CoachStatus({ coachKey }: { coachKey?: CoachKey }) {
  if (!coachKey) return <span className="ds-badge ds-badge--warn">Coach off</span>;
  if (!coachKey.enabled) return <span className="ds-badge ds-badge--warn">Paused</span>;
  return <span className="ds-badge ds-badge--ok">● Active</span>;
}

/**
 * FeatureDefaultRow is one "Default model per feature" row: what answers for this feature
 * now, and a Change (or Choose, when nothing is set yet) that opens F12's catalog list
 * scoped to this feature. Picking writes the feature's default — it never touches the key
 * itself, which is why switching the interview brain can't disturb the chat coach.
 */
function FeatureDefaultRow({
  feature,
  label,
  note,
  keys,
  current,
  catalog,
}: {
  feature: CoachFeature;
  label: string;
  note?: string;
  keys: CoachKey[];
  current: CoachFeatureDefault | null;
  catalog?: CoachModelsResponse;
}) {
  const put = usePutCoachKey();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  usePopoverDismiss(ref, open, () => setOpen(false));

  const groups = featureModelGroups(keys, catalog, feature, current);
  const anyProvider = current?.provider ?? groups[0]?.provider ?? "";

  return (
    <div className="xl-feat">
      <span className="xl-feat__k">{label}</span>
      <span className="xl-feat__v">
        {current ? (
          <>
            <b>{coachModelLabel(current.model, catalog?.models)}</b>
            <span className="ds-chip ds-chip--xs ds-mono">{current.model}</span>
            <span>· {providerLabel(current.provider)}</span>
          </>
        ) : (
          <b>Not set</b>
        )}
        {note && <span>{note}</span>}
        {put.isError && <span style={{ color: "var(--ds-err)" }}>Couldn’t save — check the key/model and try again.</span>}
      </span>
      <div ref={ref} style={{ position: "relative" }}>
        <button
          type="button"
          className="ds-btn ds-btn--secondary ds-btn--sm"
          aria-haspopup="listbox"
          aria-expanded={open}
          disabled={groups.length === 0 || put.isPending}
          onClick={() => setOpen((o) => !o)}
        >
          {current ? "Change" : "Choose"}
        </button>
        {open && (
          <CoachModelMenu
            groups={groups}
            asOf={catalog?.as_of}
            selected={current?.model ?? ""}
            customProvider={anyProvider}
            ariaLabel={`${label} model`}
            onPick={(provider, model) => {
              put.mutate({ provider, default: true, feature, default_model: model });
              setOpen(false);
            }}
          />
        )}
      </div>
    </div>
  );
}

/**
 * featureModelGroups lists the models this feature may be pointed at, grouped by connected
 * provider: every catalog chat model for `coach`, and only the interview_brain ones for
 * `interview` (a known id without that capability is refused server-side, so offering it
 * would be offering a dead end). The model currently in use is always kept in its group,
 * even when it is a custom id or one the catalog has since dropped.
 */
function featureModelGroups(keys: CoachKey[], catalog: CoachModelsResponse | undefined, feature: CoachFeature, current: CoachFeatureDefault | null): CoachModelGroup[] {
  return COACH_PROVIDERS.filter((p) => keys.some((k) => k.provider === p.id)).map((p) => {
    const all = catalogModelsFor(catalog?.models, p.id, current?.provider === p.id ? current.model : "");
    const models = feature === "interview" ? all.filter((m) => m.capabilities.includes("interview_brain") || m.id === current?.model) : all;
    return { provider: p.id, models };
  });
}

/** featureRoleTags maps provider → the feature chips its panel wears. */
function featureRoleTags(data?: CoachKeyResponse): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const { feature, tag } of COACH_ROLE_TAGS) {
    const d = coachFeatureDefault(data, feature);
    if (d) (out[d.provider] ??= []).push(tag);
  }
  return out;
}

/** UsageMonthHint is F13's month-to-date line. Display only, and always an estimate — the
 *  "custom models not estimated" tail is what keeps the number from reading as a total. */
function UsageMonthHint({ usage }: { usage: CoachUsageMonth }) {
  return (
    <div className="xl-set-hint">
      <Icon name="chart" className="xl-ico--sm" style={{ color: "var(--ds-teal)" }} />
      <span>
        This month on your keys: <b style={{ color: "var(--ds-text)" }}>{coachUsageLine(usage)}</b>
        {usage.has_unknown_cost && " · custom models not estimated"}
      </span>
    </div>
  );
}

/** One provider's panel: connect it, replace its key, or remove it. The model lives in the
 *  per-feature defaults above, not here. */
function ProviderPanel({ providerId, current, roles }: { providerId: ProviderId; current?: CoachKey; roles: string[] }) {
  const put = usePutCoachKey();
  const del = useDeleteCoachKey();
  const meta = COACH_PROVIDERS.find((p) => p.id === providerId)!;
  const connected = !!current;

  const [rawKey, setRawKey] = useState("");
  const [editingKey, setEditingKey] = useState(!connected);
  const canSave = rawKey.trim().length > 0 && !put.isPending;

  const save = () => {
    if (!canSave) return;
    // Carry the model the key already had: a PUT with a key but no default_model resets the
    // model to the catalog default server-side, which would silently undo the learner's
    // pick every time they rotated a key. A first connect has nothing to carry, so the
    // server picks the catalog default for it.
    const body: PutCoachKeyBody = { provider: providerId, key: rawKey.trim() };
    if (current?.default_model) body.default_model = current.default_model;
    put.mutate(body, {
      onSuccess: () => {
        setRawKey("");
        setEditingKey(false);
      },
    });
  };

  return (
    <div className="xl-prov">
      <div className="xl-prov__head">
        <ProviderLogo provider={providerId} size={42} />
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ fontSize: 14.5, fontWeight: 650 }}>{meta.label}</div>
          {connected ? (
            <div className="xl-prov__status">
              <span className="xl-dot" /> Connected{" "}
              <span className="ds-mono" style={{ color: "var(--ds-muted)" }}>
                · <span>{current!.masked_key}</span>
              </span>
            </div>
          ) : (
            <div className="xl-prov__status xl-prov__status--off">Not connected</div>
          )}
        </div>
        {roles.map((r) => (
          <span key={r} className="xl-tag">
            {r}
          </span>
        ))}
      </div>

      <div className="xl-keyline">
        {connected && !editingKey ? (
          <>
            <input className="ds-input ds-input--mono" value={current!.masked_key} readOnly aria-label={`${meta.label} API key`} style={{ color: "var(--ds-dim)" }} />
            <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => setEditingKey(true)}>
              Update
            </button>
          </>
        ) : (
          <input
            className="ds-input ds-input--mono"
            type="password"
            placeholder={meta.keyHint}
            aria-label={`${meta.label} API key`}
            autoComplete="off"
            spellCheck={false}
            value={rawKey}
            onChange={(e) => setRawKey(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && save()}
          />
        )}
      </div>

      <div className="xl-prov__actions">
        {(!connected || editingKey) && (
          <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled={!canSave} onClick={save}>
            <Icon name="check" className="xl-ico--sm" /> {put.isPending ? "Saving…" : connected ? `Save ${meta.label}` : `Connect ${meta.label}`}
          </button>
        )}
        {connected && (
          <button type="button" className="ds-btn ds-btn--ghost ds-btn--sm" style={{ color: "var(--ds-err)" }} disabled={del.isPending} onClick={() => del.mutate(providerId)}>
            Remove
          </button>
        )}
        <span style={{ marginLeft: "auto", fontSize: 11.5, color: "var(--ds-muted)", display: "inline-flex", alignItems: "center", gap: 6 }}>
          <Icon name="lock" className="xl-ico--sm" style={{ color: "var(--ds-ok)" }} /> Encrypted at rest
        </span>
      </div>
      {put.isError && <span style={{ fontSize: 12, color: "var(--ds-err)" }}>Couldn’t save — check the key/model and try again.</span>}
    </div>
  );
}

// --- Reminders ---

function RemindersSection({ account }: { account: Account }) {
  const save = usePatchMe();
  const [dailyOn, setDailyOn] = useState(account.reminders.daily_reminder_on ?? true);
  const [dailyTime, setDailyTime] = useState(account.reminders.daily_reminder_time ?? "20:00");
  const [alertsOn, setAlertsOn] = useState(account.reminders.revision_due_alerts_on ?? true);
  const timeValid = /^([01]\d|2[0-3]):[0-5]\d$/.test(dailyTime);

  const touch = () => {
    if (save.isSuccess || save.isError) save.reset();
  };

  return (
    <Card icon="bell" title="Reminders" subtitle="A gentle nudge, only when it helps.">
      <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
        <ToggleRow
          title="Daily study reminder"
          desc="A nudge if you haven’t started today’s plan"
          on={dailyOn}
          onToggle={() => {
            touch();
            setDailyOn((v) => !v);
          }}
        />
        {dailyOn && (
          <Field label="Remind me at" htmlFor="rem-time" style={{ maxWidth: 200, marginLeft: 4 }}>
            <input
              id="rem-time"
              className={timeValid ? "ds-input ds-input--mono" : "ds-input ds-input--mono ds-input--invalid"}
              value={dailyTime}
              onChange={(e) => {
                touch();
                setDailyTime(e.target.value);
              }}
              placeholder="20:00"
              inputMode="numeric"
            />
            {!timeValid && <span className="ds-field__error">Use 24-hour HH:MM (e.g. 20:00).</span>}
          </Field>
        )}
        <div style={{ borderTop: "1px solid var(--ds-line)", paddingTop: 14 }}>
          <ToggleRow
            title="Revision-due alerts"
            desc="When reviews pile up past your daily budget"
            on={alertsOn}
            onToggle={() => {
              touch();
              setAlertsOn((v) => !v);
            }}
          />
        </div>

        <SaveRow
          save={save}
          disabled={!timeValid}
          onSave={() => save.mutate({ reminders: { daily_reminder_on: dailyOn, daily_reminder_time: dailyTime, revision_due_alerts_on: alertsOn } })}
        />
      </div>
    </Card>
  );
}

function ToggleRow({ title, desc, on, onToggle }: { title: string; desc: string; on: boolean; onToggle: () => void }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <span style={{ flex: 1 }}>
        <b style={{ fontSize: 13 }}>{title}</b>
        <div style={{ fontSize: 11.5, color: "var(--ds-muted)" }}>{desc}</div>
      </span>
      <button type="button" role="switch" aria-checked={on} aria-label={title} className={on ? "ds-toggle ds-toggle--on" : "ds-toggle"} onClick={onToggle} style={{ background: "none", border: "none", padding: 0 }}>
        <span className="ds-toggle__track">
          <span className="ds-toggle__knob" />
        </span>
      </button>
    </div>
  );
}

// --- shared bits ---

function Card({ icon, title, subtitle, right, children }: { icon: IconName; title: string; subtitle?: string; right?: ReactNode; children: ReactNode }) {
  return (
    <section className="xl-set-card">
      <div className="xl-set-card__h">
        <span className="xl-set-card__ic">
          <Icon name={icon} className="xl-ico--sm" />
        </span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <h3 style={{ fontSize: 14.5, fontWeight: 650 }}>{title}</h3>
          {subtitle && <p style={{ fontSize: 12, color: "var(--ds-muted)", marginTop: 2 }}>{subtitle}</p>}
        </div>
        {right}
      </div>
      <div className="xl-set-card__b">{children}</div>
    </section>
  );
}

function Field({ label, htmlFor, hint, style, children }: { label: string; htmlFor?: string; hint?: string; style?: React.CSSProperties; children: ReactNode }) {
  return (
    <div className="ds-field" style={style}>
      <label className="ds-field__label" htmlFor={htmlFor}>
        {label}
        {hint && <span style={{ color: "var(--ds-muted)", fontWeight: 400, marginLeft: 6 }}>· {hint}</span>}
      </label>
      {children}
    </div>
  );
}

function SaveRow({ save, onSave, disabled }: { save: { isPending: boolean; isSuccess: boolean; isError: boolean }; onSave: () => void; disabled?: boolean }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
      <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled={disabled || save.isPending} onClick={onSave}>
        {save.isPending ? "Saving…" : "Save"}
      </button>
      {save.isSuccess && (
        <span style={{ fontSize: 12, color: "var(--ds-ok)", display: "inline-flex", alignItems: "center", gap: 5 }}>
          <Icon name="check" className="xl-ico--sm" /> Saved
        </span>
      )}
      {save.isError && <span style={{ fontSize: 12, color: "var(--ds-err)" }}>Couldn’t save — try again.</span>}
    </div>
  );
}

function Centered({ children }: { children: string }) {
  return (
    <div style={{ display: "flex", alignItems: "center", gap: 10, padding: "24px 0" }}>
      <Spinner label={children} />
    </div>
  );
}
