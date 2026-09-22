import { useEffect, useState, type ReactNode } from "react";
import { Icon, type IconName } from "../components/Icon";
import { ProviderLogo } from "../components/ProviderLogo";
import { Spinner } from "../components/States";
import { BudgetFields } from "../components/BudgetFields";
import { DEFAULT_WEEKDAY, DEFAULT_WEEKEND, budgetEta, clampWeekday, weekendLabel } from "../lib/budget";
import { useMe, usePatchMe, type Account, type WeekendBand } from "../lib/auth";
import {
  COACH_MODELS,
  COACH_PROVIDERS,
  coachModelLabel,
  providerLabel,
  useCoachKey,
  useDeleteCoachKey,
  usePutCoachKey,
  type CoachKey,
  type ProviderId,
  type PutCoachKeyBody,
} from "../lib/settings";

/**
 * Settings is the account surface: Profile, Study budget, your Coach and Reminders.
 * Redesigned (F006): a two-column layout — a sticky section rail + a column of soft cards.
 * The Coach section connects one key PER PROVIDER (Anthropic and/or OpenAI), each with its
 * own model, and one marked the default the coach answers with. Fully-rounded (pill)
 * controls + circular tiles. Every write is live (PATCH /me · PUT/DELETE /coach/key).
 */
export default function Settings() {
  const me = useMe();

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
          <SettingsRail account={me.data.account} />
          <div className="xl-settings__content">
            <section id="budget">
              <BudgetSection account={me.data.account} />
            </section>
            <section id="coach">
              <CoachSection />
            </section>
            <section id="reminders">
              <RemindersSection account={me.data.account} />
            </section>
          </div>
        </div>
      )}
    </div>
  );
}

// --- Section rail (sticky nav + identity) ---

const RAIL: { id: string; label: string; icon: IconName }[] = [
  { id: "budget", label: "Study budget", icon: "clock" },
  { id: "coach", label: "Your AI coach", icon: "spark" },
  { id: "reminders", label: "Reminders", icon: "bell" },
];

function SettingsRail({ account }: { account: Account }) {
  const [active, setActive] = useState(RAIL[0]!.id);

  useEffect(() => {
    if (typeof IntersectionObserver === "undefined") return;
    const obs = new IntersectionObserver(
      (entries) => {
        const vis = entries.filter((e) => e.isIntersecting).sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (vis[0]) setActive(vis[0].target.id);
      },
      { rootMargin: "-15% 0px -75% 0px", threshold: 0 },
    );
    RAIL.forEach(({ id }) => {
      const el = document.getElementById(id);
      if (el) obs.observe(el);
    });
    return () => obs.disconnect();
  }, []);

  const go = (id: string) => document.getElementById(id)?.scrollIntoView?.({ behavior: "smooth", block: "start" });

  return (
    <aside className="xl-set-rail">
      <ProfileCard account={account} />
      <nav className="xl-set-nav" aria-label="Settings sections">
        {RAIL.map((s) => (
          <button key={s.id} type="button" className={active === s.id ? "xl-rail xl-rail--on" : "xl-rail"} aria-current={active === s.id} onClick={() => go(s.id)}>
            <Icon name={s.icon} className="xl-ico--sm" /> {s.label}
          </button>
        ))}
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

// --- Coach (per-provider keys + default model) ---

const CUSTOM = "__custom__";

function CoachSection() {
  const coach = useCoachKey();
  const keys = coach.data?.keys ?? [];
  const defaultKey = keys.find((k) => k.is_default);
  const byProvider = (id: ProviderId) => keys.find((k) => k.provider === id);

  return (
    <Card
      icon="spark"
      title="Your AI coach"
      subtitle="Connect one or both providers. Your default model is the one the coach answers with — stored encrypted."
      right={<CoachStatus defaultKey={defaultKey} />}
    >
      {coach.isLoading ? (
        <Centered>Loading…</Centered>
      ) : (
        <div style={{ display: "flex", flexDirection: "column", gap: 14 }}>
          <div className="xl-coach-summary">
            {defaultKey ? (
              <>
                <Star /> Default: <b style={{ color: "var(--ds-text)" }}>{defaultKey.name || coachModelLabel(defaultKey.default_model)}</b>
                <span className="ds-chip ds-chip--xs ds-mono">{defaultKey.default_model}</span>
                <span style={{ color: "var(--ds-muted)" }}>· {providerLabel(defaultKey.provider)}</span>
              </>
            ) : (
              <span style={{ color: "var(--ds-muted)" }}>No default yet — connect a provider below to start coaching.</span>
            )}
          </div>
          {COACH_PROVIDERS.map((p) => {
            const cur = byProvider(p.id);
            // Remount on connect/disconnect so the panel's local edit state (model/name/key
            // field) re-seeds from the freshly-connected key rather than the empty form.
            return <ProviderPanel key={`${p.id}-${cur ? "on" : "off"}`} providerId={p.id} current={cur} />;
          })}
        </div>
      )}
    </Card>
  );
}

function CoachStatus({ defaultKey }: { defaultKey?: CoachKey }) {
  if (!defaultKey) return <span className="ds-badge ds-badge--warn">Coach off</span>;
  if (!defaultKey.enabled) return <span className="ds-badge ds-badge--warn">Paused</span>;
  return <span className="ds-badge ds-badge--ok">● Active</span>;
}

/** One provider's panel — connect it, pick its model, name it, and mark it the default.
 *  Handles both the connected and the not-yet-connected states. */
function ProviderPanel({ providerId, current }: { providerId: ProviderId; current?: CoachKey }) {
  const put = usePutCoachKey();
  const del = useDeleteCoachKey();
  const meta = COACH_PROVIDERS.find((p) => p.id === providerId)!;
  const models = COACH_MODELS[providerId];
  const connected = !!current;

  const inList = current ? models.some((m) => m.id === current.default_model) : true;
  const [modelSel, setModelSel] = useState<string>(current && !inList ? CUSTOM : current?.default_model ?? models[0]!.id);
  const [customModel, setCustomModel] = useState(current && !inList ? current.default_model : "");
  const [name, setName] = useState(current?.name ?? "");
  const [rawKey, setRawKey] = useState("");
  const [editingKey, setEditingKey] = useState(!connected);

  const model = modelSel === CUSTOM ? customModel.trim() : modelSel;
  const canSave = model.length > 0 && (connected || rawKey.trim().length > 0) && !put.isPending;

  const save = () => {
    if (!canSave) return;
    const finalName = name.trim() || coachModelLabel(model);
    const body: PutCoachKeyBody = rawKey.trim()
      ? { provider: providerId, key: rawKey.trim(), default_model: model, name: finalName }
      : { provider: providerId, default_model: model, name: finalName };
    put.mutate(body, {
      onSuccess: () => {
        setRawKey("");
        setEditingKey(false);
      },
    });
  };

  return (
    <div className={connected && current!.is_default ? "xl-prov xl-prov--default" : "xl-prov"}>
      {/* header */}
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
        {connected &&
          (current!.is_default ? (
            <span className="xl-defbadge">
              <Star size={12} /> Default
            </span>
          ) : (
            <button type="button" className="xl-setdef" disabled={put.isPending} onClick={() => put.mutate({ provider: providerId, default: true })}>
              <Star size={13} /> Set as default
            </button>
          ))}
      </div>

      {/* model pills */}
      <Field label="Model">
        <div className="xl-pills">
          {models.map((m) => (
            <button
              key={m.id}
              type="button"
              className={modelSel === m.id ? "xl-mpill xl-mpill--on" : "xl-mpill"}
              aria-pressed={modelSel === m.id}
              onClick={() => {
                setModelSel(m.id);
                setCustomModel("");
              }}
            >
              {m.label}
            </button>
          ))}
          <button type="button" className={modelSel === CUSTOM ? "xl-mpill xl-mpill--on" : "xl-mpill"} aria-pressed={modelSel === CUSTOM} onClick={() => setModelSel(CUSTOM)}>
            Custom…
          </button>
        </div>
        {modelSel === CUSTOM && (
          <input
            className="ds-input ds-input--mono"
            style={{ marginTop: 10 }}
            placeholder={providerId === "anthropic" ? "claude-…" : "gpt-…"}
            value={customModel}
            onChange={(e) => setCustomModel(e.target.value)}
          />
        )}
      </Field>

      {/* name + key */}
      <div className="xl-prov__grid">
        <Field label="Name" hint="default = model">
          <input className="ds-input" placeholder={coachModelLabel(model) || "My coach"} value={name} onChange={(e) => setName(e.target.value)} />
        </Field>
        <Field label="API key">
          {connected && !editingKey ? (
            <div className="xl-keyline">
              <input className="ds-input ds-input--mono" value={current!.masked_key} readOnly style={{ color: "var(--ds-dim)" }} />
              <button type="button" className="ds-btn ds-btn--secondary ds-btn--sm" onClick={() => setEditingKey(true)}>
                Update
              </button>
            </div>
          ) : (
            <input
              className="ds-input ds-input--mono"
              type="password"
              placeholder={meta.keyHint}
              value={rawKey}
              onChange={(e) => setRawKey(e.target.value)}
              onKeyDown={(e) => e.key === "Enter" && save()}
            />
          )}
        </Field>
      </div>

      {/* actions */}
      <div className="xl-prov__actions">
        <button type="button" className="ds-btn ds-btn--primary ds-btn--sm" disabled={!canSave} onClick={save}>
          <Icon name="check" className="xl-ico--sm" /> {put.isPending ? "Saving…" : connected ? `Save ${meta.label}` : `Connect ${meta.label}`}
        </button>
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

/** A small filled star for the default marker (the Icon set has no star). */
function Star({ size = 12 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden="true" style={{ flex: "none" }}>
      <path d="M12 2l2.9 6.3L22 9.3l-5 4.9 1.2 7L12 17.8 5.8 21.2 7 14.2 2 9.3l7.1-1z" />
    </svg>
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
